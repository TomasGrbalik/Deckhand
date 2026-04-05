package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/service"
)

func TestHumanizeError_nil(t *testing.T) {
	if got := humanizeError(nil); got != "" {
		t.Fatalf("expected empty string for nil error, got %q", got)
	}
}

func TestHumanizeError_dockerConnectionRefused(t *testing.T) {
	err := fmt.Errorf("connecting to docker: %w",
		fmt.Errorf("pinging docker daemon: %w",
			errors.New("connection refused")))

	got := humanizeError(err)
	assertContains(t, got, "connection refused")
	assertContains(t, got, "Is Docker running?")
}

func TestHumanizeError_dockerCannotConnect(t *testing.T) {
	err := errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock")

	got := humanizeError(err)
	assertContains(t, got, "Cannot connect to the Docker daemon")
	assertContains(t, got, "Is Docker running?")
}

func TestHumanizeError_dockerPermissionDenied(t *testing.T) {
	err := errors.New("permission denied while trying to connect to the docker daemon socket")

	got := humanizeError(err)
	assertContains(t, got, "Is Docker running?")
}

func TestHumanizeError_missingProjectConfig(t *testing.T) {
	err := fmt.Errorf("loading config: %w",
		fmt.Errorf("reading file: %w", fs.ErrNotExist))

	got := humanizeError(err)
	assertContains(t, got, "loading config")
	assertContains(t, got, "deckhand init")
}

func TestHumanizeError_noEnvironment(t *testing.T) {
	err := fmt.Errorf("stopping environment: %w", service.ErrNoEnvironment)

	got := humanizeError(err)
	assertContains(t, got, "no environment found")
	assertContains(t, got, "deckhand up")
}

func TestHumanizeError_templateNotFound(t *testing.T) {
	err := fmt.Errorf("loading template metadata: %w",
		fmt.Errorf("template %q: %w", "custom", fs.ErrNotExist))

	got := humanizeError(err)
	assertContains(t, got, "template")
	assertContains(t, got, "deckhand template list")
}

func TestHumanizeError_unknownError(t *testing.T) {
	err := errors.New("something unexpected happened")

	got := humanizeError(err)
	if got != "something unexpected happened" {
		t.Fatalf("expected original message unchanged, got %q", got)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !contains(got, want) {
		t.Errorf("expected output to contain %q, got:\n%s", want, got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
