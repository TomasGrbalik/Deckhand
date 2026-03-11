package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client for daemon communication.
type Client struct {
	api client.APIClient
}

// NewClient creates a Docker client using the default environment settings
// (DOCKER_HOST, DOCKER_TLS_VERIFY, etc.).
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return &Client{api: cli}, nil
}

// Ping verifies the Docker daemon is reachable.
func (c *Client) Ping(ctx context.Context) (types.Ping, error) {
	resp, err := c.api.Ping(ctx)
	if err != nil {
		return types.Ping{}, fmt.Errorf("pinging docker daemon: %w", err)
	}
	return resp, nil
}

// Close releases the underlying Docker client resources.
func (c *Client) Close() error {
	return c.api.Close()
}

// API returns the underlying Docker API client for use by other infra
// components (compose, container).
func (c *Client) API() client.APIClient {
	return c.api
}
