package docker_test

import (
	"context"
	"strings"
	"testing"

	dockervolume "github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"

	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
)

// newTestDockerAPI returns a real Docker API client, skipping if Docker is unavailable.
func newTestDockerAPI(t *testing.T) dockerclient.APIClient {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli.API()
}

func TestVolume_ListByProject_Empty(t *testing.T) {
	api := newTestDockerAPI(t)
	vol := docker.NewVolume(api)

	// Use a project name unlikely to exist.
	vols, err := vol.ListByProject("deckhand-test-nonexistent-project-xyz")
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(vols) != 0 {
		t.Errorf("expected 0 volumes, got %d", len(vols))
	}
}

func TestVolume_ListAndRemove(t *testing.T) {
	api := newTestDockerAPI(t)
	vol := docker.NewVolume(api)
	ctx := context.Background()

	projectName := "deckhand-test-vol-" + strings.ReplaceAll(t.Name(), "/", "-")
	volName := projectName + "-workspace"

	// Create a labeled volume via the Docker API directly.
	_, err := api.VolumeCreate(ctx, dockervolume.CreateOptions{
		Name: volName,
		Labels: map[string]string{
			"dev.deckhand.managed": "true",
			"dev.deckhand.project": projectName,
			"dev.deckhand.volume":  "workspace",
		},
	})
	if err != nil {
		t.Fatalf("creating test volume: %v", err)
	}
	t.Cleanup(func() { _ = api.VolumeRemove(ctx, volName, true) })

	// ListByProject should find it.
	vols, err := vol.ListByProject(projectName)
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(vols) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(vols))
	}
	if vols[0].Name != volName {
		t.Errorf("expected volume name %q, got %q", volName, vols[0].Name)
	}

	// Remove should succeed.
	err = vol.Remove(volName)
	if err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	// After removal, list should be empty.
	vols, err = vol.ListByProject(projectName)
	if err != nil {
		t.Fatalf("ListByProject() after remove error: %v", err)
	}
	if len(vols) != 0 {
		t.Errorf("expected 0 volumes after remove, got %d", len(vols))
	}
}
