package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// VolumeInfo holds metadata about a Docker volume discovered by label.
type VolumeInfo struct {
	Name string
	Size int64 // size in bytes, -1 if unknown
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

	f := client.Filters{}
	f = f.Add("label", "dev.deckhand.managed=true")
	f = f.Add("label", "dev.deckhand.project="+projectName)

	resp, err := v.api.VolumeList(ctx, client.VolumeListOptions{Filters: f})
	if err != nil {
		return nil, fmt.Errorf("listing volumes for project %q: %w", projectName, err)
	}

	result := make([]VolumeInfo, 0, len(resp.Items))
	for _, vol := range resp.Items {
		size := int64(-1)
		if vol.UsageData != nil {
			size = vol.UsageData.Size
		}
		result = append(result, VolumeInfo{Name: vol.Name, Size: size})
	}
	return result, nil
}

// Remove deletes a volume by name.
func (v *Volume) Remove(volumeName string) error {
	ctx := context.Background()
	if _, err := v.api.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{}); err != nil {
		return fmt.Errorf("removing volume %q: %w", volumeName, err)
	}
	return nil
}
