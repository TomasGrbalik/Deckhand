package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/config"
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
