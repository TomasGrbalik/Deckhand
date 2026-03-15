package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Compose shells out to the docker compose CLI for orchestration.
// The CLI handles multi-container orchestration better than the SDK.
type Compose struct{}

// NewCompose creates a Compose runner.
func NewCompose() *Compose {
	return &Compose{}
}

// Up starts containers defined in the compose file.
// If build is true, images are rebuilt before starting.
func (c *Compose) Up(projectDir, composePath string, build bool) error {
	args := []string{"compose", "-f", composePath, "up", "-d"}
	if build {
		args = append(args, "--build")
	}
	return c.run(projectDir, args...)
}

// Down stops and removes containers defined in the compose file.
func (c *Compose) Down(projectDir, composePath string) error {
	return c.run(projectDir, "compose", "-f", composePath, "down")
}

// Destroy stops containers and removes volumes and orphans.
func (c *Compose) Destroy(projectDir, composePath string) error {
	return c.run(projectDir, "compose", "-f", composePath, "down", "-v", "--remove-orphans")
}

func (c *Compose) run(dir string, args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Dir = dir

	// Capture stderr for error reporting while also streaming it to the
	// user so they can see build progress in real time.
	var stderrBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %v: %w\n%s", args, err, stderrBuf.String())
	}
	return nil
}
