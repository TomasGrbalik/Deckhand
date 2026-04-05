package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ComposeVersion runs "docker compose version --short" and returns the
// trimmed version string (e.g. "2.24.0").
func (c *Compose) ComposeVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "compose", "version", "--short").Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("docker compose version timed out")
		}
		return "", fmt.Errorf("docker compose not available: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
