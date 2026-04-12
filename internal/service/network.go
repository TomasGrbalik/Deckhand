package service

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"

	goyaml "gopkg.in/yaml.v3"
)

// NetworkState tracks which project has which IP on the shared network.
// Persisted at ~/.config/deckhand/network-state.yaml.
type NetworkState struct {
	Assignments map[string]string `yaml:"assignments"`
}

// LoadNetworkState reads the network state file. Returns an empty state
// (not an error) if the file does not exist.
func LoadNetworkState(path string) (*NetworkState, error) {
	state := &NetworkState{Assignments: make(map[string]string)}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return nil, fmt.Errorf("reading network state %s: %w", path, err)
	}

	if err := goyaml.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parsing network state %s: %w", path, err)
	}

	if state.Assignments == nil {
		state.Assignments = make(map[string]string)
	}

	return state, nil
}

// SaveNetworkState writes the network state file, creating parent
// directories if needed.
func SaveNetworkState(path string, state *NetworkState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := goyaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling network state: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing network state %s: %w", path, err)
	}

	return nil
}

// ipOffset is the first host offset used for allocation (.10).
const ipOffset = 10

// AllocateIP returns the existing IP for the project, or assigns the next
// free one starting at .10 in the configured subnet. The state is mutated
// in place — the caller must call SaveNetworkState afterward.
func AllocateIP(state *NetworkState, subnet, projectName string) (string, error) {
	if ip, ok := state.Assignments[projectName]; ok {
		return ip, nil
	}

	prefix, err := netip.ParsePrefix(subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet %q: %w", subnet, err)
	}

	used := make(map[netip.Addr]bool, len(state.Assignments))
	for _, ip := range state.Assignments {
		addr, parseErr := netip.ParseAddr(ip)
		if parseErr == nil {
			used[addr] = true
		}
	}

	// Start at base + ipOffset (e.g., 172.30.0.10).
	candidate := prefix.Addr()
	for range ipOffset {
		candidate = candidate.Next()
	}

	for prefix.Contains(candidate) {
		if !used[candidate] {
			state.Assignments[projectName] = candidate.String()
			return candidate.String(), nil
		}
		candidate = candidate.Next()
	}

	return "", fmt.Errorf("no free IPs in subnet %s", subnet)
}

// FreeIP removes a project's IP assignment from the state.
// The caller must call SaveNetworkState afterward.
func FreeIP(state *NetworkState, projectName string) {
	delete(state.Assignments, projectName)
}

// ProjectIP returns the assigned IP for a project, or empty string if none.
func ProjectIP(state *NetworkState, projectName string) string {
	return state.Assignments[projectName]
}
