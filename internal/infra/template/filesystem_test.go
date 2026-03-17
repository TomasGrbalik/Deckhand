package template_test

import (
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
