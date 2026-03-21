package domain

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestVolumeMount_YAMLRoundTrip(t *testing.T) {
	input := `
mounts:
  volumes:
    - name: workspace
      target: /workspace
`
	var p Project
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(p.Mounts.Volumes) != 1 {
		t.Fatalf("Volumes length: got %d, want 1", len(p.Mounts.Volumes))
	}
	v := p.Mounts.Volumes[0]
	if v.Name != "workspace" {
		t.Errorf("Name: got %q, want %q", v.Name, "workspace")
	}
	if v.Target != "/workspace" {
		t.Errorf("Target: got %q, want %q", v.Target, "/workspace")
	}
	if v.Enabled != nil {
		t.Errorf("Enabled: got %v, want nil", v.Enabled)
	}
}

func TestSecretMount_YAMLRoundTrip(t *testing.T) {
	input := `
mounts:
  secrets:
    - name: gh-token
      source: "${GH_TOKEN}"
      env: GH_TOKEN
`
	var p Project
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(p.Mounts.Secrets) != 1 {
		t.Fatalf("Secrets length: got %d, want 1", len(p.Mounts.Secrets))
	}
	s := p.Mounts.Secrets[0]
	if s.Source != "${GH_TOKEN}" {
		t.Errorf("Source: got %q, want %q", s.Source, "${GH_TOKEN}")
	}
	if s.Env != "GH_TOKEN" {
		t.Errorf("Env: got %q, want %q", s.Env, "GH_TOKEN")
	}
	if s.Target != "" {
		t.Errorf("Target: got %q, want empty", s.Target)
	}
}

func TestSecretMount_EnabledFalse(t *testing.T) {
	input := `
mounts:
  secrets:
    - name: gh-token
      enabled: false
`
	var p Project
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := p.Mounts.Secrets[0]
	if s.Enabled == nil {
		t.Fatal("Enabled: got nil, want pointer to false")
	}
	if *s.Enabled != false {
		t.Errorf("Enabled: got %v, want false", *s.Enabled)
	}
}

func TestSecretMount_Validate_NoOutput(t *testing.T) {
	s := SecretMount{
		Name:   "gh-token",
		Source: "${GH_TOKEN}",
	}
	if err := s.Validate(); err == nil {
		t.Error("expected validation error for secret with no env and no target")
	}
}

func TestSecretMount_Validate_WithEnv(t *testing.T) {
	s := SecretMount{
		Name:   "gh-token",
		Source: "${GH_TOKEN}",
		Env:    "GH_TOKEN",
	}
	if err := s.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSecretMount_Validate_WithTarget(t *testing.T) {
	s := SecretMount{
		Name:   "gitconfig",
		Source: "~/.gitconfig",
		Target: "/home/dev/.gitconfig",
	}
	if err := s.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTemplateMeta_MountsYAML(t *testing.T) {
	input := `
name: go
description: Go dev environment
variables:
  go_version:
    default: "1.22"
    description: Go version
mounts:
  volumes:
    - name: workspace
      target: /workspace
    - name: go-mod-cache
      target: /home/dev/go/pkg/mod
`
	var m TemplateMeta
	if err := yaml.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(m.Mounts.Volumes) != 2 {
		t.Fatalf("Volumes length: got %d, want 2", len(m.Mounts.Volumes))
	}
	if m.Mounts.Volumes[0].Name != "workspace" {
		t.Errorf("Volumes[0].Name: got %q, want %q", m.Mounts.Volumes[0].Name, "workspace")
	}
}

func TestMounts_OmitEmpty(t *testing.T) {
	p := Project{
		Name:     "test",
		Template: "base",
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if contains(string(data), "mounts") {
		t.Errorf("marshaled YAML should not contain 'mounts' when empty\nGot:\n%s", data)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFullProjectWithMounts_YAMLRoundTrip(t *testing.T) {
	input := `
project: my-api
template: go
mounts:
  volumes:
    - name: workspace
      target: /workspace
  secrets:
    - name: gh-token
      source: "${GH_TOKEN}"
      env: GH_TOKEN
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
  sockets:
    - name: ssh-agent
      source: "${SSH_AUTH_SOCK}"
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK
`
	var p Project
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(p.Mounts.Volumes) != 1 {
		t.Fatalf("Volumes: got %d, want 1", len(p.Mounts.Volumes))
	}
	if len(p.Mounts.Secrets) != 2 {
		t.Fatalf("Secrets: got %d, want 2", len(p.Mounts.Secrets))
	}
	if len(p.Mounts.Sockets) != 1 {
		t.Fatalf("Sockets: got %d, want 1", len(p.Mounts.Sockets))
	}

	// Verify secret fields
	gitconfig := p.Mounts.Secrets[1]
	if gitconfig.ReadOnly != true {
		t.Errorf("gitconfig.ReadOnly: got %v, want true", gitconfig.ReadOnly)
	}
	if gitconfig.Target != "/home/dev/.gitconfig" {
		t.Errorf("gitconfig.Target: got %q, want %q", gitconfig.Target, "/home/dev/.gitconfig")
	}

	// Verify socket fields
	ssh := p.Mounts.Sockets[0]
	if ssh.Source != "${SSH_AUTH_SOCK}" {
		t.Errorf("ssh.Source: got %q, want %q", ssh.Source, "${SSH_AUTH_SOCK}")
	}
	if ssh.Env != "SSH_AUTH_SOCK" {
		t.Errorf("ssh.Env: got %q, want %q", ssh.Env, "SSH_AUTH_SOCK")
	}
}
