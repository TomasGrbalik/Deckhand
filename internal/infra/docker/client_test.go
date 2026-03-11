package docker_test

import (
	"context"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
)

func TestNewClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer cli.Close()

	resp, err := cli.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}

	if resp.APIVersion == "" {
		t.Error("expected non-empty API version from ping")
	}
}
