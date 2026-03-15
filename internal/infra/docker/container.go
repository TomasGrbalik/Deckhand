package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/term"
)

// Container uses the Docker SDK for container-level operations
// (exec, logs, find). The SDK is needed here for proper TTY handling
// and streaming.
type Container struct {
	api client.APIClient
}

// NewContainer creates a Container operator using the given Docker API client.
func NewContainer(api client.APIClient) *Container {
	return &Container{api: api}
}

// Exec runs a command inside a container, identified by name or ID.
// When tty is true, stdin is attached and the terminal is put into raw mode
// for interactive use.
func (c *Container) Exec(containerName string, cmd []string, tty bool) error {
	ctx := context.Background()

	execCfg := container.ExecOptions{
		Cmd:          cmd,
		AttachStdin:  tty,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          tty,
	}

	execID, err := c.api.ContainerExecCreate(ctx, containerName, execCfg)
	if err != nil {
		return fmt.Errorf("creating exec in %q: %w", containerName, err)
	}

	resp, err := c.api.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{Tty: tty})
	if err != nil {
		return fmt.Errorf("attaching to exec in %q: %w", containerName, err)
	}
	defer resp.Close()

	if tty {
		return c.execInteractive(ctx, resp, execID.ID)
	}
	return c.execNonInteractive(resp)
}

func (c *Container) execInteractive(_ context.Context, resp types.HijackedResponse, _ string) error {
	//nolint:gosec // Fd() returns uintptr; converting to int is safe for terminal FDs on all supported platforms.
	fd := int(os.Stdin.Fd())
	oldState, termErr := term.MakeRaw(fd)
	if termErr != nil {
		return fmt.Errorf("setting terminal raw mode: %w", termErr)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	outputDone := make(chan error, 1)
	inputDone := make(chan error, 1)

	// Copy container output to stdout.
	go func() {
		_, copyErr := io.Copy(os.Stdout, resp.Reader)
		outputDone <- copyErr
	}()

	// Copy stdin to container.
	go func() {
		_, copyErr := io.Copy(resp.Conn, os.Stdin)
		inputDone <- copyErr
	}()

	// Wait for the output goroutine to finish (container command exited),
	// then close the connection to unblock the stdin goroutine.
	outputErr := <-outputDone
	resp.Close()
	<-inputDone

	if outputErr != nil {
		return fmt.Errorf("exec stream: %w", outputErr)
	}

	return nil
}

func (c *Container) execNonInteractive(resp types.HijackedResponse) error {
	if _, err := io.Copy(os.Stdout, resp.Reader); err != nil {
		return fmt.Errorf("reading exec output: %w", err)
	}
	return nil
}

// Logs returns a reader streaming the container's logs.
func (c *Container) Logs(containerName string, follow bool, tail string) (io.ReadCloser, error) {
	ctx := context.Background()

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
	}

	rc, err := c.api.ContainerLogs(ctx, containerName, opts)
	if err != nil {
		return nil, fmt.Errorf("getting logs for %q: %w", containerName, err)
	}
	return rc, nil
}

// FindContainer locates a container by deckhand labels:
// dev.deckhand.project=<projectName> and dev.deckhand.service=<serviceName>.
// Returns the container ID or an error if not found.
func (c *Container) FindContainer(projectName, serviceName string) (string, error) {
	ctx := context.Background()

	f := filters.NewArgs(
		filters.Arg("label", "dev.deckhand.project="+projectName),
		filters.Arg("label", "dev.deckhand.service="+serviceName),
	)

	containers, err := c.api.ContainerList(ctx, container.ListOptions{Filters: f})
	if err != nil {
		return "", fmt.Errorf("listing containers: %w", err)
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("container not found for project %q service %q", projectName, serviceName)
	}

	return containers[0].ID, nil
}
