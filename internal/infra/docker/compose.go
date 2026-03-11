package docker

import (
	"fmt"
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
	cmd.Stdout = nil
	cmd.Stderr = nil

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker %v: %w\n%s", args, err, out)
	}
	return nil
}
