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
