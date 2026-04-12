package docker_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
)

const testNetworkName = "deckhand-test-net"

// createTestNetwork creates a Docker network for testing and returns a cleanup function.
func createTestNetwork(t *testing.T) func() {
	t.Helper()

	// Remove any leftover network from a previous run.
	_ = exec.Command("docker", "network", "rm", testNetworkName).Run()

	cmd := exec.Command("docker", "network", "create",
		"--driver=bridge",
		"--subnet=172.31.255.0/24",
		testNetworkName,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("creating test network: %v\n%s", err, out)
	}

	return func() {
		_ = exec.Command("docker", "network", "rm", testNetworkName).Run()
	}
}

func TestNetworkExists_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanup := createTestNetwork(t)
	defer cleanup()

	client, err := docker.NewClient(context.Background())
	if err != nil {
		t.Fatalf("creating docker client: %v", err)
	}
	defer client.Close()

	net := docker.NewNetwork(client.API())
	exists, err := net.NetworkExists(testNetworkName)
	if err != nil {
		t.Fatalf("NetworkExists() error: %v", err)
	}
	if !exists {
		t.Error("expected network to exist")
	}
}

func TestNetworkExists_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client, err := docker.NewClient(context.Background())
	if err != nil {
		t.Fatalf("creating docker client: %v", err)
	}
	defer client.Close()

	net := docker.NewNetwork(client.API())
	exists, err := net.NetworkExists("deckhand-nonexistent-net-" + t.Name())
	if err != nil {
		t.Fatalf("NetworkExists() error: %v", err)
	}
	if exists {
		t.Error("expected network to not exist")
	}
}
