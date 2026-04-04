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
// (DOCKER_HOST, DOCKER_TLS_VERIFY, etc.).
func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
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
