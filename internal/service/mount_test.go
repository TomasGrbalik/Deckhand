package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

func TestMergeMounts_BasicThreeWayMerge(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_abc123")
	t.Setenv("API_KEY", "key_xyz")

	template := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "workspace", Target: "/workspace"},
		},
	}
	global := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "gh-token", Source: "${GH_TOKEN}", Env: "GH_TOKEN"},
		},
	}
	project := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "api-key", Source: "${API_KEY}", Env: "API_KEY"},
		},
	}

	result, warnings := MergeMounts(template, global, project)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(result.Volumes))
	}
	if result.Volumes[0].Name != "workspace" {
		t.Errorf("expected volume name workspace, got %s", result.Volumes[0].Name)
	}
	if len(result.Secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(result.Secrets))
	}
	if result.Secrets[0].Name != "gh-token" || result.Secrets[0].Source != "ghp_abc123" {
		t.Errorf("unexpected gh-token secret: %+v", result.Secrets[0])
	}
	if result.Secrets[1].Name != "api-key" || result.Secrets[1].Source != "key_xyz" {
		t.Errorf("unexpected api-key secret: %+v", result.Secrets[1])
	}
}

func TestMergeMounts_ProjectOverridesTemplateDefault(t *testing.T) {
	template := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "workspace", Target: "/workspace"},
		},
	}
	project := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "workspace", Target: "/home/dev/code"},
		},
	}

	result, warnings := MergeMounts(template, domain.Mounts{}, project)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(result.Volumes))
	}
	if result.Volumes[0].Target != "/home/dev/code" {
		t.Errorf("expected target /home/dev/code, got %s", result.Volumes[0].Target)
	}
}

func TestMergeMounts_EnabledFalseRemovesInheritedMount(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_abc123")

	global := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "gh-token", Source: "${GH_TOKEN}", Env: "GH_TOKEN"},
		},
	}
	project := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "gh-token", Enabled: new(bool)},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, global, project)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Secrets) != 0 {
		t.Errorf("expected no secrets, got %d: %+v", len(result.Secrets), result.Secrets)
	}
}

func TestMergeMounts_EnvVarResolution(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_abc123")

	input := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "gh-token", Source: "${GH_TOKEN}", Env: "GH_TOKEN"},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, input)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(result.Secrets))
	}
	if result.Secrets[0].Source != "ghp_abc123" {
		t.Errorf("expected resolved source ghp_abc123, got %s", result.Secrets[0].Source)
	}
}

func TestMergeMounts_UnresolvableEnvVar(t *testing.T) {
	t.Setenv("UNSET_VAR", "")

	input := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "missing-token", Source: "${UNSET_VAR}", Env: "TOKEN"},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, input)

	if len(result.Secrets) != 0 {
		t.Errorf("expected secret to be skipped, got %d secrets", len(result.Secrets))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0] == "" {
		t.Error("expected non-empty warning")
	}
}

func TestMergeMounts_TildeExpansion(t *testing.T) {
	// Create a temp file to simulate ~/.gitconfig.
	tmpDir := t.TempDir()
	fakeHome := tmpDir
	gitconfig := filepath.Join(fakeHome, ".gitconfig")
	if err := os.WriteFile(gitconfig, []byte("[user]\nname = test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override HOME so ~ expands to our temp dir.
	t.Setenv("HOME", fakeHome)

	input := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "gitconfig", Source: "~/.gitconfig", Target: "/home/dev/.gitconfig"},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, input)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(result.Secrets))
	}
	expected := filepath.Join(fakeHome, ".gitconfig")
	if result.Secrets[0].Source != expected {
		t.Errorf("expected source %s, got %s", expected, result.Secrets[0].Source)
	}
}

func TestMergeMounts_MissingSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	input := domain.Mounts{
		Secrets: []domain.SecretMount{
			{Name: "missing-file", Source: "~/.nonexistent-file", Target: "/home/dev/.nonexistent"},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, input)

	if len(result.Secrets) != 0 {
		t.Errorf("expected secret to be skipped, got %d secrets", len(result.Secrets))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0] == "" {
		t.Error("expected non-empty warning")
	}
}

func TestMergeMounts_EmptyInputs(t *testing.T) {
	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, domain.Mounts{})

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Volumes) != 0 || len(result.Secrets) != 0 || len(result.Sockets) != 0 {
		t.Errorf("expected empty mounts, got %+v", result)
	}
}

func TestMergeMounts_SocketEnvVarResolution(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "agent.sock")
	if err := os.WriteFile(sockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSH_AUTH_SOCK", sockPath)

	input := domain.Mounts{
		Sockets: []domain.SocketMount{
			{Name: "ssh-agent", Source: "${SSH_AUTH_SOCK}", Target: "/run/ssh-agent.sock", Env: "SSH_AUTH_SOCK"},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, input)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(result.Sockets) != 1 {
		t.Fatalf("expected 1 socket, got %d", len(result.Sockets))
	}
	if result.Sockets[0].Source != sockPath {
		t.Errorf("expected source %s, got %s", sockPath, result.Sockets[0].Source)
	}
}

func TestMergeMounts_SocketUnresolvableEnvVar(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	input := domain.Mounts{
		Sockets: []domain.SocketMount{
			{Name: "ssh-agent", Source: "${SSH_AUTH_SOCK}", Target: "/run/ssh-agent.sock"},
		},
	}

	result, warnings := MergeMounts(domain.Mounts{}, domain.Mounts{}, input)

	if len(result.Sockets) != 0 {
		t.Errorf("expected socket to be skipped, got %d sockets", len(result.Sockets))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestMergeMounts_PreservesInsertionOrder(t *testing.T) {
	template := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "workspace", Target: "/workspace"},
			{Name: "cache", Target: "/cache"},
		},
	}
	global := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "data", Target: "/data"},
		},
	}
	project := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "logs", Target: "/logs"},
		},
	}

	result, _ := MergeMounts(template, global, project)

	if len(result.Volumes) != 4 {
		t.Fatalf("expected 4 volumes, got %d", len(result.Volumes))
	}
	expected := []string{"workspace", "cache", "data", "logs"}
	for i, name := range expected {
		if result.Volumes[i].Name != name {
			t.Errorf("volume[%d]: expected %s, got %s", i, name, result.Volumes[i].Name)
		}
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~", home},
		{"~/foo/bar", home + "/foo/bar"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := expandTilde(tt.input)
		if got != tt.expected {
			t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
