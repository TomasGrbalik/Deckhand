package docker

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/client"
)

// Network checks Docker network state.
type Network struct {
	api client.APIClient
}

// NewNetwork creates a Network checker using the given Docker API client.
func NewNetwork(api client.APIClient) *Network {
	return &Network{api: api}
}

// NetworkExists returns true if a Docker network with the given name exists.
func (n *Network) NetworkExists(name string) (bool, error) {
	ctx := context.Background()

	_, err := n.api.NetworkInspect(ctx, name, client.NetworkInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspecting network %q: %w", name, err)
	}

	return true, nil
}
