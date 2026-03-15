package docker_test

import (
	"context"
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

	name = "deckhand-test-" + project + "-" + service
	// Remove any leftover container from a previous run.
	_ = exec.Command("docker", "rm", "-f", name).Run()

	cmd := exec.Command("docker", "run", "-d",
		"--name", name,
		"--label", "dev.deckhand.managed=true",
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

func TestListByProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	_, cleanup := testContainer(t, "testlist", "devcontainer")
	defer cleanup()

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	containers, err := ctr.ListByProject("testlist")
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}

	if len(containers) == 0 {
		t.Fatal("expected at least 1 container")
	}

	c := containers[0]
	if c.Project != "testlist" {
		t.Errorf("Project = %q, want %q", c.Project, "testlist")
	}
	if c.Service != "devcontainer" {
		t.Errorf("Service = %q, want %q", c.Service, "devcontainer")
	}
	if c.State != "running" {
		t.Errorf("State = %q, want %q", c.State, "running")
	}
	if c.Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", c.Image, "alpine:latest")
	}
}

func TestListByProjectEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	containers, err := ctr.ListByProject("nonexistent-project-xyz")
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(containers) != 0 {
		t.Errorf("expected 0 containers, got %d", len(containers))
	}
}

func TestListAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	_, cleanup1 := testContainer(t, "testall-a", "devcontainer")
	defer cleanup1()
	_, cleanup2 := testContainer(t, "testall-b", "devcontainer")
	defer cleanup2()

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	defer cli.Close()

	ctr := docker.NewContainer(cli.API())
	containers, err := ctr.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error: %v", err)
	}

	// Should find at least our two test containers.
	projects := make(map[string]bool)
	for _, c := range containers {
		projects[c.Project] = true
	}
	if !projects["testall-a"] {
		t.Error("ListAll() missing project testall-a")
	}
	if !projects["testall-b"] {
		t.Error("ListAll() missing project testall-b")
	}
}

func TestLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Start a container that produces output.
	name := "deckhand-test-logs"
	// Remove any leftover container from a previous run.
	_ = exec.Command("docker", "rm", "-f", name).Run()
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
