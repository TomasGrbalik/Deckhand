package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

// ComposeVersion runs "docker compose version --short" and returns the
// trimmed version string (e.g. "2.24.0").
func (c *Compose) ComposeVersion() (string, error) {
	out, err := exec.Command("docker", "compose", "version", "--short").Output()
	if err != nil {
		return "", fmt.Errorf("docker compose not available: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
