package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// Client wraps the Docker SDK client for daemon communication.
type Client struct {
	api client.APIClient
}

// NewClient creates a Docker client using the default environment settings
// (DOCKER_HOST, DOCKER_TLS_VERIFY, etc.) and negotiates the API version
// with the Docker daemon via Ping.
func NewClient(ctx context.Context) (*Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	// Negotiate API version with the daemon — the new Moby client requires
	// an explicit Ping call (previously handled by WithAPIVersionNegotiation).
	if _, err := cli.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true}); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("pinging docker daemon: %w", err)
	}
	return &Client{api: cli}, nil
}

// Ping verifies the Docker daemon is reachable and negotiates the API version.
func (c *Client) Ping(ctx context.Context) (client.PingResult, error) {
	resp, err := c.api.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true})
	if err != nil {
		return client.PingResult{}, fmt.Errorf("pinging docker daemon: %w", err)
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
