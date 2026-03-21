package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// VolumeInfo holds metadata about a Docker volume discovered by label.
type VolumeInfo struct {
	Name string
}

// Volume uses the Docker SDK to list and remove named volumes by label.
type Volume struct {
	api client.APIClient
}

// NewVolume creates a Volume operator using the given Docker API client.
func NewVolume(api client.APIClient) *Volume {
	return &Volume{api: api}
}

// ListByProject returns all volumes labeled with dev.deckhand.project=<name>.
func (v *Volume) ListByProject(projectName string) ([]VolumeInfo, error) {
	ctx := context.Background()

	f := filters.NewArgs(
		filters.Arg("label", "dev.deckhand.managed=true"),
		filters.Arg("label", "dev.deckhand.project="+projectName),
	)

	resp, err := v.api.VolumeList(ctx, volume.ListOptions{Filters: f})
	if err != nil {
		return nil, fmt.Errorf("listing volumes for project %q: %w", projectName, err)
	}

	result := make([]VolumeInfo, 0, len(resp.Volumes))
	for _, vol := range resp.Volumes {
		result = append(result, VolumeInfo{Name: vol.Name})
	}
	return result, nil
}

// Remove deletes a volume by name.
func (v *Volume) Remove(volumeName string) error {
	ctx := context.Background()
	if err := v.api.VolumeRemove(ctx, volumeName, false); err != nil {
		return fmt.Errorf("removing volume %q: %w", volumeName, err)
	}
	return nil
}
