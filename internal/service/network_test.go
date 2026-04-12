package service_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/service"
)

func TestAllocateIP_FirstProject(t *testing.T) {
	state := &service.NetworkState{Assignments: map[string]string{}}

	ip, err := service.AllocateIP(state, "172.30.0.0/24", "myapp")
	if err != nil {
		t.Fatalf("AllocateIP() error: %v", err)
	}

	if ip != "172.30.0.10" {
		t.Errorf("expected 172.30.0.10, got %s", ip)
	}

	if state.Assignments["myapp"] != "172.30.0.10" {
		t.Errorf("state not updated, got %v", state.Assignments)
	}
}

func TestAllocateIP_SecondProject(t *testing.T) {
	state := &service.NetworkState{
		Assignments: map[string]string{
			"first": "172.30.0.10",
		},
	}

	ip, err := service.AllocateIP(state, "172.30.0.0/24", "second")
	if err != nil {
		t.Fatalf("AllocateIP() error: %v", err)
	}

	if ip != "172.30.0.11" {
		t.Errorf("expected 172.30.0.11, got %s", ip)
	}
}

func TestAllocateIP_ReusesExistingIP(t *testing.T) {
	state := &service.NetworkState{
		Assignments: map[string]string{
			"myapp": "172.30.0.10",
		},
	}

	ip, err := service.AllocateIP(state, "172.30.0.0/24", "myapp")
	if err != nil {
		t.Fatalf("AllocateIP() error: %v", err)
	}

	if ip != "172.30.0.10" {
		t.Errorf("expected reused IP 172.30.0.10, got %s", ip)
	}
}

func TestAllocateIP_SkipsUsedIPs(t *testing.T) {
	state := &service.NetworkState{
		Assignments: map[string]string{
			"a": "172.30.0.10",
			"b": "172.30.0.11",
			"c": "172.30.0.12",
		},
	}

	ip, err := service.AllocateIP(state, "172.30.0.0/24", "d")
	if err != nil {
		t.Fatalf("AllocateIP() error: %v", err)
	}

	if ip != "172.30.0.13" {
		t.Errorf("expected 172.30.0.13, got %s", ip)
	}
}

func TestAllocateIP_FillsGaps(t *testing.T) {
	state := &service.NetworkState{
		Assignments: map[string]string{
			"a": "172.30.0.10",
			// 172.30.0.11 is free (was freed)
			"c": "172.30.0.12",
		},
	}

	ip, err := service.AllocateIP(state, "172.30.0.0/24", "new")
	if err != nil {
		t.Fatalf("AllocateIP() error: %v", err)
	}

	if ip != "172.30.0.11" {
		t.Errorf("expected gap-fill 172.30.0.11, got %s", ip)
	}
}

func TestAllocateIP_SkipsBroadcast(t *testing.T) {
	// /30 subnet: 172.30.0.0, .1 (gw), .2, .3 (broadcast).
	// Starting at .10 offset won't work for /30, so use a larger subnet
	// but fill up to just before broadcast.
	// Use /28: 172.30.0.0 - 172.30.0.15 (broadcast = .15).
	state := &service.NetworkState{Assignments: map[string]string{}}
	// Fill .10 through .14
	for i := 10; i <= 14; i++ {
		state.Assignments[fmt.Sprintf("proj%d", i)] = fmt.Sprintf("172.30.0.%d", i)
	}
	// Next candidate would be .15 (broadcast) — should be skipped.
	_, err := service.AllocateIP(state, "172.30.0.0/28", "new")
	if err == nil {
		t.Fatal("expected error — no free IPs (broadcast should be excluded)")
	}
}

func TestAllocateIP_InvalidSubnet(t *testing.T) {
	state := &service.NetworkState{Assignments: map[string]string{}}

	_, err := service.AllocateIP(state, "not-a-subnet", "myapp")
	if err == nil {
		t.Fatal("expected error for invalid subnet")
	}
}

func TestFreeIP(t *testing.T) {
	state := &service.NetworkState{
		Assignments: map[string]string{
			"myapp": "172.30.0.10",
			"other": "172.30.0.11",
		},
	}

	service.FreeIP(state, "myapp")

	if _, ok := state.Assignments["myapp"]; ok {
		t.Error("myapp should be removed from assignments")
	}
	if state.Assignments["other"] != "172.30.0.11" {
		t.Error("other project should be unchanged")
	}
}

func TestFreeIP_NonexistentProject(_ *testing.T) {
	state := &service.NetworkState{Assignments: map[string]string{}}

	// Should not panic.
	service.FreeIP(state, "nonexistent")
}

func TestProjectIP(t *testing.T) {
	state := &service.NetworkState{
		Assignments: map[string]string{
			"myapp": "172.30.0.10",
		},
	}

	if ip := service.ProjectIP(state, "myapp"); ip != "172.30.0.10" {
		t.Errorf("expected 172.30.0.10, got %s", ip)
	}
	if ip := service.ProjectIP(state, "unknown"); ip != "" {
		t.Errorf("expected empty string for unknown project, got %s", ip)
	}
}

func TestLoadNetworkState_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")

	state, err := service.LoadNetworkState(path)
	if err != nil {
		t.Fatalf("LoadNetworkState() error: %v", err)
	}

	if state.Assignments == nil {
		t.Fatal("assignments map should be initialized")
	}
	if len(state.Assignments) != 0 {
		t.Errorf("expected empty assignments, got %v", state.Assignments)
	}
}

func TestLoadAndSaveNetworkState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.yaml")

	state := &service.NetworkState{
		Assignments: map[string]string{
			"myapp": "172.30.0.10",
		},
	}

	if err := service.SaveNetworkState(path, state); err != nil {
		t.Fatalf("SaveNetworkState() error: %v", err)
	}

	loaded, err := service.LoadNetworkState(path)
	if err != nil {
		t.Fatalf("LoadNetworkState() error: %v", err)
	}

	if loaded.Assignments["myapp"] != "172.30.0.10" {
		t.Errorf("expected myapp=172.30.0.10, got %v", loaded.Assignments)
	}
}

func TestSaveNetworkState_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "state.yaml")

	state := &service.NetworkState{Assignments: map[string]string{}}
	if err := service.SaveNetworkState(path, state); err != nil {
		t.Fatalf("SaveNetworkState() error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file should exist: %v", err)
	}
}
