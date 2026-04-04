package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/config"
	"github.com/TomasGrbalik/deckhand/internal/domain"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: base
ports:
  - port: 8080
    name: web
    protocol: http
    internal: false
  - port: 5432
    name: postgres
    protocol: tcp
    internal: true
env:
  GO_ENV: development
  DATABASE_URL: postgres://localhost/mydb
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Name != "myapp" {
		t.Errorf("Name = %q, want %q", proj.Name, "myapp")
	}
	if proj.Template != "base" {
		t.Errorf("Template = %q, want %q", proj.Template, "base")
	}
	if len(proj.Ports) != 2 {
		t.Fatalf("len(Ports) = %d, want 2", len(proj.Ports))
	}
	if proj.Ports[0].Port != 8080 {
		t.Errorf("Ports[0].Port = %d, want 8080", proj.Ports[0].Port)
	}
	if proj.Ports[0].Name != "web" {
		t.Errorf("Ports[0].Name = %q, want %q", proj.Ports[0].Name, "web")
	}
	if proj.Ports[1].Internal != true {
		t.Errorf("Ports[1].Internal = %v, want true", proj.Ports[1].Internal)
	}
	if len(proj.Env) != 2 {
		t.Fatalf("len(Env) = %d, want 2", len(proj.Env))
	}
	if proj.Env["GO_ENV"] != "development" {
		t.Errorf("Env[GO_ENV] = %q, want %q", proj.Env["GO_ENV"], "development")
	}
}

func TestLoad_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Version != 1 {
		t.Errorf("Version = %d, want 1 (default for missing version)", proj.Version)
	}
}

func TestLoad_Version1(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `version: 1
project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Version != 1 {
		t.Errorf("Version = %d, want 1", proj.Version)
	}
}

func TestLoad_UnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `version: 2
project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(cfgPath)
	if err == nil {
		t.Fatal("Load() should return error for unsupported version")
	}

	want := "config version 2 is not supported — please upgrade deckhand"
	if got := err.Error(); got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}

func TestSave_SetsVersionWhenZero(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	proj := &domain.Project{
		Name:     "myapp",
		Template: "base",
		// Version is 0 (zero value)
	}

	if err := config.Save(cfgPath, proj); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Read the raw file to verify version is present.
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	if !containsStr(string(data), "version: 1") {
		t.Errorf("saved file should contain 'version: 1', got:\n%s", data)
	}

	// Also verify round-trip.
	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/.deckhand.yaml")
	if err == nil {
		t.Fatal("Load() should return error for missing file")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	// Write invalid YAML (tabs are not allowed as indentation in YAML)
	content := "project: myapp\n\t\tinvalid:\nyaml: [[[broken"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(cfgPath)
	if err == nil {
		t.Fatal("Load() should return error for invalid YAML")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	original := &domain.Project{
		Version:  1,
		Name:     "myapp",
		Template: "base",
		Ports: []domain.PortMapping{
			{Port: 8080, Name: "web", Protocol: "http"},
			{Port: 5432, Name: "pg", Protocol: "tcp", Internal: true},
		},
		Env: map[string]string{"GO_ENV": "dev"},
	}

	if err := config.Save(cfgPath, original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
	if len(loaded.Ports) != 2 {
		t.Fatalf("len(Ports) = %d, want 2", len(loaded.Ports))
	}
	if loaded.Ports[0].Port != 8080 {
		t.Errorf("Ports[0].Port = %d, want 8080", loaded.Ports[0].Port)
	}
	if loaded.Ports[1].Internal != true {
		t.Errorf("Ports[1].Internal = %v, want true", loaded.Ports[1].Internal)
	}
	if loaded.Env["GO_ENV"] != "dev" {
		t.Errorf("Env[GO_ENV] = %q, want %q", loaded.Env["GO_ENV"], "dev")
	}
}

func TestSave_RoundTrip_WithVariables(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	original := &domain.Project{
		Name:      "myapp",
		Template:  "go",
		Variables: map[string]string{"go_version": "1.22", "lint": "true"},
	}

	if err := config.Save(cfgPath, original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}

	if len(loaded.Variables) != 2 {
		t.Fatalf("len(Variables) = %d, want 2", len(loaded.Variables))
	}
	if loaded.Variables["go_version"] != "1.22" {
		t.Errorf("Variables[go_version] = %q, want %q", loaded.Variables["go_version"], "1.22")
	}
	if loaded.Variables["lint"] != "true" {
		t.Errorf("Variables[lint] = %q, want %q", loaded.Variables["lint"], "true")
	}
}

func TestLoad_ConfigWithVariables(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: go
variables:
  go_version: "1.22"
  lint: "true"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if len(proj.Variables) != 2 {
		t.Fatalf("len(Variables) = %d, want 2", len(proj.Variables))
	}
	if proj.Variables["go_version"] != "1.22" {
		t.Errorf("Variables[go_version] = %q, want %q", proj.Variables["go_version"], "1.22")
	}
	if proj.Variables["lint"] != "true" {
		t.Errorf("Variables[lint] = %q, want %q", proj.Variables["lint"], "true")
	}
}

func TestLoad_ConfigWithoutVariables(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if len(proj.Variables) != 0 {
		t.Errorf("Variables should be empty, got %v", proj.Variables)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Name != "" {
		t.Errorf("Name = %q, want empty", proj.Name)
	}
	if proj.Template != "" {
		t.Errorf("Template = %q, want empty", proj.Template)
	}
	if len(proj.Ports) != 0 {
		t.Errorf("len(Ports) = %d, want 0", len(proj.Ports))
	}
	if len(proj.Env) != 0 {
		t.Errorf("len(Env) = %d, want 0", len(proj.Env))
	}
}

func TestLoad_EnvVarOverrideProject(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DECKHAND_PROJECT", "override-name")

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Name != "override-name" {
		t.Errorf("Name = %q, want %q", proj.Name, "override-name")
	}
	if proj.Template != "base" {
		t.Errorf("Template = %q, want %q (should be unchanged)", proj.Template, "base")
	}
}

func TestLoad_EnvVarOverrideTemplate(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: go
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DECKHAND_TEMPLATE", "rust")

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Template != "rust" {
		t.Errorf("Template = %q, want %q", proj.Template, "rust")
	}
	if proj.Name != "myapp" {
		t.Errorf("Name = %q, want %q (should be unchanged)", proj.Name, "myapp")
	}
}

func TestLoad_EnvVarOverrideBoth(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DECKHAND_PROJECT", "ci-build")
	t.Setenv("DECKHAND_TEMPLATE", "node")

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Name != "ci-build" {
		t.Errorf("Name = %q, want %q", proj.Name, "ci-build")
	}
	if proj.Template != "node" {
		t.Errorf("Template = %q, want %q", proj.Template, "node")
	}
}

func TestLoad_EnvVarEmptyDoesNotOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".deckhand.yaml")

	content := `project: myapp
template: base
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DECKHAND_PROJECT", "")
	t.Setenv("DECKHAND_TEMPLATE", "")

	proj, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if proj.Name != "myapp" {
		t.Errorf("Name = %q, want %q (empty env var should not override)", proj.Name, "myapp")
	}
	if proj.Template != "base" {
		t.Errorf("Template = %q, want %q (empty env var should not override)", proj.Template, "base")
	}
}

func TestLoadGlobal_CompleteConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `defaults:
  template: go
  shell: zsh

ssh:
  user: dev
  host: 100.64.1.3

mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}
      env: GH_TOKEN
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
  sockets:
    - name: ssh-agent
      source: ${SSH_AUTH_SOCK}
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadGlobal(cfgPath)
	if err != nil {
		t.Fatalf("LoadGlobal() returned error: %v", err)
	}

	if cfg.Defaults.Template != "go" {
		t.Errorf("Defaults.Template = %q, want %q", cfg.Defaults.Template, "go")
	}
	if cfg.Defaults.Shell != "zsh" {
		t.Errorf("Defaults.Shell = %q, want %q", cfg.Defaults.Shell, "zsh")
	}
	if cfg.SSH.User != "dev" {
		t.Errorf("SSH.User = %q, want %q", cfg.SSH.User, "dev")
	}
	if cfg.SSH.Host != "100.64.1.3" {
		t.Errorf("SSH.Host = %q, want %q", cfg.SSH.Host, "100.64.1.3")
	}
	if len(cfg.Mounts.Secrets) != 2 {
		t.Fatalf("len(Mounts.Secrets) = %d, want 2", len(cfg.Mounts.Secrets))
	}
	if cfg.Mounts.Secrets[0].Name != "gh-token" {
		t.Errorf("Mounts.Secrets[0].Name = %q, want %q", cfg.Mounts.Secrets[0].Name, "gh-token")
	}
	if cfg.Mounts.Secrets[1].ReadOnly != true {
		t.Errorf("Mounts.Secrets[1].ReadOnly = %v, want true", cfg.Mounts.Secrets[1].ReadOnly)
	}
	if len(cfg.Mounts.Sockets) != 1 {
		t.Fatalf("len(Mounts.Sockets) = %d, want 1", len(cfg.Mounts.Sockets))
	}
	if cfg.Mounts.Sockets[0].Name != "ssh-agent" {
		t.Errorf("Mounts.Sockets[0].Name = %q, want %q", cfg.Mounts.Sockets[0].Name, "ssh-agent")
	}
}

func TestLoadGlobal_MissingFile(t *testing.T) {
	cfg, err := config.LoadGlobal("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("LoadGlobal() should not return error for missing file, got: %v", err)
	}

	// Should return zero-value GlobalConfig.
	if cfg.Defaults.Template != "" {
		t.Errorf("Defaults.Template = %q, want empty", cfg.Defaults.Template)
	}
	if cfg.Defaults.Shell != "" {
		t.Errorf("Defaults.Shell = %q, want empty", cfg.Defaults.Shell)
	}
	if cfg.SSH.User != "" {
		t.Errorf("SSH.User = %q, want empty", cfg.SSH.User)
	}
	if len(cfg.Mounts.Volumes) != 0 {
		t.Errorf("len(Mounts.Volumes) = %d, want 0", len(cfg.Mounts.Volumes))
	}
}

func TestLoadGlobal_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `defaults:
  template: go
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadGlobal(cfgPath)
	if err != nil {
		t.Fatalf("LoadGlobal() returned error: %v", err)
	}

	if cfg.Defaults.Template != "go" {
		t.Errorf("Defaults.Template = %q, want %q", cfg.Defaults.Template, "go")
	}
	if cfg.Defaults.Shell != "" {
		t.Errorf("Defaults.Shell = %q, want empty", cfg.Defaults.Shell)
	}
	if cfg.SSH.User != "" {
		t.Errorf("SSH.User = %q, want empty", cfg.SSH.User)
	}
	if len(cfg.Mounts.Secrets) != 0 {
		t.Errorf("len(Mounts.Secrets) = %d, want 0", len(cfg.Mounts.Secrets))
	}
}

func TestLoadGlobal_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := "defaults:\n\t\tinvalid:\nyaml: [[[broken"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadGlobal(cfgPath)
	if err == nil {
		t.Fatal("LoadGlobal() should return error for malformed YAML")
	}
}
