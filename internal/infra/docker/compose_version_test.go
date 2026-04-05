package docker_test

import (
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
)

func TestComposeVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	comp := docker.NewCompose()
	version, err := comp.ComposeVersion()
	if err != nil {
		t.Fatalf("ComposeVersion() error: %v", err)
	}
	if version == "" {
		t.Error("ComposeVersion() returned empty string")
	}
}
