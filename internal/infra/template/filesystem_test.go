package template_test

import (
	"errors"
	ioFs "io/fs"
	"os"
	"path/filepath"
	"testing"

	tmpl "github.com/TomasGrbalik/deckhand/internal/infra/template"
)

func TestFilesystemSource_List_WithValidTemplate(t *testing.T) {
	dir := t.TempDir()

	// Create a valid template.
	rustDir := filepath.Join(dir, "rust")
	if err := os.Mkdir(rustDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rustDir, "metadata.yaml"), []byte("name: rust\ndescription: Rust dev container\nvariables: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	result, err := fs.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 template, got %d", len(result))
	}
	if result[0].Name != "rust" {
		t.Errorf("Name = %q, want %q", result[0].Name, "rust")
	}
	if result[0].Source != "user" {
		t.Errorf("Source = %q, want %q", result[0].Source, "user")
	}
	if result[0].Description != "Rust dev container" {
		t.Errorf("Description = %q, want %q", result[0].Description, "Rust dev container")
	}
}

func TestFilesystemSource_List_NonexistentDir(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "does-not-exist")
	fs := &tmpl.FilesystemSource{Dir: missingDir}
	result, err := fs.List()
	if err != nil {
		t.Fatalf("List() should not error for nonexistent dir, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 templates, got %d", len(result))
	}
}

func TestFilesystemSource_List_SkipsMissingMetadata(t *testing.T) {
	dir := t.TempDir()

	// Template dir without metadata.yaml — should be skipped.
	if err := os.Mkdir(filepath.Join(dir, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	result, err := fs.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 templates (broken skipped), got %d", len(result))
	}
}

func TestFilesystemSource_Load(t *testing.T) {
	dir := t.TempDir()
	name := "mytemplate"
	tmplDir := filepath.Join(dir, name)
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "Dockerfile.tmpl"), []byte("FROM ubuntu"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "compose.yaml.tmpl"), []byte("services: {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	dockerfile, compose, err := fs.Load(name)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if dockerfile != "FROM ubuntu" {
		t.Errorf("Dockerfile = %q, want %q", dockerfile, "FROM ubuntu")
	}
	if compose != "services: {}" {
		t.Errorf("compose = %q, want %q", compose, "services: {}")
	}
}

func TestFilesystemSource_LoadMeta(t *testing.T) {
	dir := t.TempDir()
	name := "mytemplate"
	tmplDir := filepath.Join(dir, name)
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "metadata.yaml"), []byte("name: mytemplate\ndescription: My template\nvariables: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	meta, err := fs.LoadMeta(name)
	if err != nil {
		t.Fatalf("LoadMeta() error: %v", err)
	}
	if meta.Name != "mytemplate" {
		t.Errorf("Name = %q, want %q", meta.Name, "mytemplate")
	}
}

func TestFilesystemSource_List_CustomSourceLabel(t *testing.T) {
	dir := t.TempDir()

	rustDir := filepath.Join(dir, "rust")
	if err := os.Mkdir(rustDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rustDir, "metadata.yaml"), []byte("name: rust\ndescription: Rust dev container\nvariables: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir, SourceLabel: "local"}
	result, err := fs.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 template, got %d", len(result))
	}
	if result[0].Source != "local" {
		t.Errorf("Source = %q, want %q", result[0].Source, "local")
	}
}

func TestFilesystemSource_List_DefaultSourceLabel(t *testing.T) {
	dir := t.TempDir()

	goDir := filepath.Join(dir, "go")
	if err := os.Mkdir(goDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "metadata.yaml"), []byte("name: go\ndescription: Go dev container\nvariables: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// No SourceLabel set — should default to "user".
	fs := &tmpl.FilesystemSource{Dir: dir}
	result, err := fs.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 template, got %d", len(result))
	}
	if result[0].Source != "user" {
		t.Errorf("Source = %q, want %q", result[0].Source, "user")
	}
}

func TestFilesystemSource_Load_MissingDirReturnsNotExist(t *testing.T) {
	fs := &tmpl.FilesystemSource{Dir: t.TempDir()}

	_, _, err := fs.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ioFs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got: %v", err)
	}
}

func TestFilesystemSource_Load_FallsBackToSharedCompose(t *testing.T) {
	dir := t.TempDir()

	// Create template dir with only Dockerfile — no compose.yaml.tmpl.
	// The loader should fall back to the shared embedded compose template.
	tmplDir := filepath.Join(dir, "partial")
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "Dockerfile.tmpl"), []byte("FROM ubuntu"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	dockerfile, compose, err := fs.Load("partial")
	if err != nil {
		t.Fatalf("Load() should succeed with shared compose fallback, got: %v", err)
	}
	if dockerfile != "FROM ubuntu" {
		t.Errorf("Dockerfile = %q, want %q", dockerfile, "FROM ubuntu")
	}
	if compose == "" {
		t.Error("compose should not be empty — shared fallback should provide it")
	}
}

func TestFilesystemSource_Load_UsesTemplateComposeOverride(t *testing.T) {
	dir := t.TempDir()

	// Create template dir with both Dockerfile and a custom compose.yaml.tmpl.
	// The loader should use the template's own compose, not the shared one.
	tmplDir := filepath.Join(dir, "custom")
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "Dockerfile.tmpl"), []byte("FROM alpine"), 0o644); err != nil {
		t.Fatal(err)
	}
	customCompose := "services:\n  custom: {}"
	if err := os.WriteFile(filepath.Join(tmplDir, "compose.yaml.tmpl"), []byte(customCompose), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	_, compose, err := fs.Load("custom")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if compose != customCompose {
		t.Errorf("compose = %q, want template-specific %q", compose, customCompose)
	}
}

func TestFilesystemSource_LoadMeta_IncompleteTemplateDoesNotReturnNotExist(t *testing.T) {
	dir := t.TempDir()

	// Create template dir without metadata.yaml.
	tmplDir := filepath.Join(dir, "partial")
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	_, err := fs.LoadMeta("partial")
	if err == nil {
		t.Fatal("expected error for missing metadata, got nil")
	}
	if errors.Is(err, ioFs.ErrNotExist) {
		t.Error("missing metadata in existing dir should not return fs.ErrNotExist")
	}
}

func TestFilesystemSource_Load_RejectsPathTraversal(t *testing.T) {
	fs := &tmpl.FilesystemSource{Dir: t.TempDir()}

	traversalNames := []string{"../../../etc/passwd", "..", "foo/bar", "/absolute"}
	for _, name := range traversalNames {
		_, _, err := fs.Load(name)
		if err == nil {
			t.Errorf("Load(%q) should return error for path traversal", name)
		}
		_, err = fs.LoadMeta(name)
		if err == nil {
			t.Errorf("LoadMeta(%q) should return error for path traversal", name)
		}
	}
}

func TestEmbeddedSource_List(t *testing.T) {
	src := &tmpl.EmbeddedSource{}
	result, err := src.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected at least one embedded template")
	}

	// The "base" template should always be present.
	found := false
	for _, info := range result {
		if info.Name == "base" {
			found = true
			if info.Source != "builtin" {
				t.Errorf("base source = %q, want %q", info.Source, "builtin")
			}
			if info.Description == "" {
				t.Error("base description should not be empty")
			}
		}
	}
	if !found {
		t.Error("base template not found in embedded listing")
	}
}

func TestFilesystemSource_LoadMeta_ParsesCommandAndExecUser(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "daemon")
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `name: daemon
description: runs a daemon
command: "/usr/sbin/sshd -D -e"
exec_user: dev
variables: {}
`
	if err := os.WriteFile(filepath.Join(tmplDir, "metadata.yaml"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	got, err := fs.LoadMeta("daemon")
	if err != nil {
		t.Fatalf("LoadMeta() error: %v", err)
	}
	if got.Command != "/usr/sbin/sshd -D -e" {
		t.Errorf("Command = %q, want %q", got.Command, "/usr/sbin/sshd -D -e")
	}
	if got.ExecUser != "dev" {
		t.Errorf("ExecUser = %q, want %q", got.ExecUser, "dev")
	}
}

func TestFilesystemSource_LoadMeta_AbsentCommandAndExecUser(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "plain")
	if err := os.Mkdir(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := "name: plain\ndescription: plain template\nvariables: {}\n"
	if err := os.WriteFile(filepath.Join(tmplDir, "metadata.yaml"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &tmpl.FilesystemSource{Dir: dir}
	got, err := fs.LoadMeta("plain")
	if err != nil {
		t.Fatalf("LoadMeta() error: %v", err)
	}
	if got.Command != "" {
		t.Errorf("Command = %q, want empty", got.Command)
	}
	if got.ExecUser != "" {
		t.Errorf("ExecUser = %q, want empty", got.ExecUser)
	}
}
