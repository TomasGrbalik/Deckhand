package docker_test

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
)

// testContainer starts an alpine container with deckhand labels and returns
// its name. The caller must call the returned cleanup function.
func testContainer(t *testing.T, project, service string) (name string, cleanup func()) {
	t.Helper()

	name = fmt.Sprintf("deckhand-test-%s-%s-%d", project, service, time.Now().UnixNano())

	cmd := exec.Command("docker", "run", "-d",
		"--name", name,
		"--label", "dev.deckhand.project="+project,
		"--label", "dev.deckhand.service="+service,
		"alpine:latest", "sleep", "infinity",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("starting test container: %v\n%s", err, out)
	}

	return name, func() {
		rm := exec.Command("docker", "rm", "-f", name)
		_ = rm.Run()
	}
}

func TestFindContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	name, cleanup := testContainer(t, "testproj", "devcontainer")
	defer cleanup()

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	id, err := ctr.FindContainer("testproj", "devcontainer")
	if err != nil {
		t.Fatalf("FindContainer() error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty container ID")
	}

	// Verify we found the right container by checking the name.
	inspect := exec.Command("docker", "inspect", "--format", "{{.Name}}", id)
	out, err := inspect.Output()
	if err != nil {
		t.Fatalf("docker inspect: %v", err)
	}
	got := strings.TrimSpace(strings.TrimPrefix(string(out), "/"))
	if got != name {
		t.Errorf("FindContainer() found %q, want %q", got, name)
	}
}

func TestFindContainerNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	_, err = ctr.FindContainer("nonexistent-project", "nonexistent-service")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestExecNonInteractive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	name, cleanup := testContainer(t, "testexec", "dev")
	defer cleanup()

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	// Run a simple command without TTY.
	err = ctr.Exec(name, []string{"echo", "hello"}, false)
	if err != nil {
		t.Fatalf("Exec() error: %v", err)
	}
}

func TestExecContainerNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	err = ctr.Exec("nonexistent-container-xyz", []string{"echo", "hello"}, false)
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
}

func TestLogsContainerNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	_, err = ctr.Logs("nonexistent-container-xyz", false, "10")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
}

func TestLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Start a container that produces output.
	name := fmt.Sprintf("deckhand-test-logs-%d", time.Now().UnixNano())
	cmd := exec.Command("docker", "run", "-d", "--name", name,
		"alpine:latest", "sh", "-c", "echo hello-from-logs && sleep infinity")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("starting test container: %v\n%s", err, out)
	}
	defer func() {
		rm := exec.Command("docker", "rm", "-f", name)
		_ = rm.Run()
	}()

	// Give the container a moment to produce output.
	time.Sleep(500 * time.Millisecond)

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	rc, err := ctr.Logs(name, false, "10")
	if err != nil {
		t.Fatalf("Logs() error: %v", err)
	}
	defer rc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(rc)
		done <- data
	}()

	select {
	case data := <-done:
		if !strings.Contains(string(data), "hello-from-logs") {
			t.Errorf("expected logs to contain 'hello-from-logs', got: %q", string(data))
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for logs")
	}
}
