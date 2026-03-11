package docker_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
)

const testComposeContent = `services:
  test:
    image: alpine:latest
    command: sleep infinity
`

func TestComposeUpDown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(testComposeContent), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}

	comp := docker.NewCompose()

	// Pull image first to avoid timeout on slow networks.
	pullCmd := exec.Command("docker", "pull", "alpine:latest")
	if out, err := pullCmd.CombinedOutput(); err != nil {
		t.Fatalf("pulling alpine: %v\n%s", err, out)
	}

	if err := comp.Up(dir, composePath, false); err != nil {
		t.Fatalf("Up() error: %v", err)
	}

	// Verify container is running.
	psCmd := exec.Command("docker", "compose", "-f", composePath, "ps", "-q")
	psCmd.Dir = dir
	out, err := psCmd.Output()
	if err != nil {
		t.Fatalf("docker compose ps: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected at least one running container after Up()")
	}

	// Clean up.
	if err := comp.Down(dir, composePath); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
}

func TestComposeDestroy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(testComposeContent), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}

	comp := docker.NewCompose()

	if err := comp.Up(dir, composePath, false); err != nil {
		t.Fatalf("Up() error: %v", err)
	}

	if err := comp.Destroy(dir, composePath); err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}

	// Verify no containers remain.
	psCmd := exec.Command("docker", "compose", "-f", composePath, "ps", "-q")
	psCmd.Dir = dir
	out, err := psCmd.Output()
	if err != nil {
		t.Fatalf("docker compose ps: %v", err)
	}
	if len(out) != 0 {
		t.Error("expected no containers after Destroy()")
	}
}
