package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/moby/moby/client"
	"golang.org/x/term"
)

// ContainerInfo holds container metadata returned by listing operations.
// This is an infra-layer type — the service layer maps it to domain.Container.
type ContainerInfo struct {
	ID      string
	Name    string
	Service string
	Project string
	Image   string
	State   string
	Status  string
	Created time.Time
	Ports   []PortInfo
}

// PortInfo represents a port mapping on a running container.
type PortInfo struct {
	Public  int
	Private int
}

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

	execCfg := client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdin:  tty,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          tty,
	}

	execID, err := c.api.ExecCreate(ctx, containerName, execCfg)
	if err != nil {
		return fmt.Errorf("creating exec in %q: %w", containerName, err)
	}

	resp, err := c.api.ExecAttach(ctx, execID.ID, client.ExecAttachOptions{TTY: tty})
	if err != nil {
		return fmt.Errorf("attaching to exec in %q: %w", containerName, err)
	}
	defer resp.Close()

	if tty {
		return c.execInteractive(ctx, resp.HijackedResponse)
	}
	return c.execNonInteractive(resp.HijackedResponse)
}

func (c *Container) execInteractive(_ context.Context, resp client.HijackedResponse) error {
	//nolint:gosec // Fd() returns uintptr; converting to int is safe for terminal FDs on all supported platforms.
	fd := int(os.Stdin.Fd())
	oldState, termErr := term.MakeRaw(fd)
	if termErr != nil {
		return fmt.Errorf("setting terminal raw mode: %w", termErr)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	outputDone := make(chan error, 1)

	// Copy container output to stdout.
	go func() {
		_, copyErr := io.Copy(os.Stdout, resp.Reader)
		outputDone <- copyErr
	}()

	// Copy stdin to container. This goroutine will block on os.Stdin.Read()
	// after the shell exits — there's no way to unblock it without closing
	// stdin itself. Following the Docker CLI's approach, we don't wait for
	// it; resp.Close() in the caller's defer will clean up the write side.
	go func() {
		_, _ = io.Copy(resp.Conn, os.Stdin)
	}()

	// Wait only for output to finish (shell exited), then return.
	if outputErr := <-outputDone; outputErr != nil {
		return fmt.Errorf("exec stream: %w", outputErr)
	}

	return nil
}

func (c *Container) execNonInteractive(resp client.HijackedResponse) error {
	if _, err := io.Copy(os.Stdout, resp.Reader); err != nil {
		return fmt.Errorf("reading exec output: %w", err)
	}
	return nil
}

// Logs returns a reader streaming the container's logs.
func (c *Container) Logs(containerName string, follow bool, tail string) (io.ReadCloser, error) {
	ctx := context.Background()

	opts := client.ContainerLogsOptions{
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

	f := client.Filters{}
	f = f.Add("label", "dev.deckhand.project="+projectName)
	f = f.Add("label", "dev.deckhand.service="+serviceName)

	result, err := c.api.ContainerList(ctx, client.ContainerListOptions{Filters: f})
	if err != nil {
		return "", fmt.Errorf("listing containers: %w", err)
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("container not found for project %q service %q", projectName, serviceName)
	}

	return result.Items[0].ID, nil
}

// ListByProject returns all deckhand-managed containers for a specific project.
func (c *Container) ListByProject(projectName string) ([]ContainerInfo, error) {
	f := client.Filters{}
	f = f.Add("label", "dev.deckhand.managed=true")
	f = f.Add("label", "dev.deckhand.project="+projectName)
	return c.listContainers(f)
}

// ListAll returns all deckhand-managed containers across all projects.
func (c *Container) ListAll() ([]ContainerInfo, error) {
	f := client.Filters{}
	f = f.Add("label", "dev.deckhand.managed=true")
	return c.listContainers(f)
}

func (c *Container) listContainers(f client.Filters) ([]ContainerInfo, error) {
	ctx := context.Background()

	// Include stopped containers so list/status show everything.
	listResult, err := c.api.ContainerList(ctx, client.ContainerListOptions{
		Filters: f,
		All:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(listResult.Items))
	for _, s := range listResult.Items {
		var ports []PortInfo
		for _, p := range s.Ports {
			if p.PublicPort != 0 {
				ports = append(ports, PortInfo{
					Public:  int(p.PublicPort),
					Private: int(p.PrivatePort),
				})
			}
		}

		name := ""
		if len(s.Names) > 0 {
			// Docker prepends "/" to container names.
			name = s.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		result = append(result, ContainerInfo{
			ID:      s.ID,
			Name:    name,
			Service: s.Labels["dev.deckhand.service"],
			Project: s.Labels["dev.deckhand.project"],
			Image:   s.Image,
			State:   string(s.State),
			Status:  s.Status,
			Created: time.Unix(s.Created, 0),
			Ports:   ports,
		})
	}

	return result, nil
}
