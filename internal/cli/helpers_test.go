package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

// fakeTemplateSource is a test double for service.TemplateSource.
type fakeTemplateSource struct {
	dockerfile string
	compose    string
	meta       *domain.TemplateMeta
	err        error
}

func (f *fakeTemplateSource) Load(string) (string, string, error) {
	return f.dockerfile, f.compose, f.err
}

func (f *fakeTemplateSource) LoadMeta(string) (*domain.TemplateMeta, error) {
	return f.meta, f.err
}

func TestCompositeSource_LocalFirstPrecedence(t *testing.T) {
	local := &fakeTemplateSource{
		dockerfile: "FROM local",
		compose:    "local compose",
	}
	user := &fakeTemplateSource{
		dockerfile: "FROM user",
		compose:    "user compose",
	}
	embedded := &fakeTemplateSource{
		dockerfile: "FROM embedded",
		compose:    "embedded compose",
	}

	cs := &compositeSource{sources: []service.TemplateSource{local, user, embedded}}
	df, comp, err := cs.Load("base")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if df != "FROM local" {
		t.Errorf("dockerfile = %q, want %q", df, "FROM local")
	}
	if comp != "local compose" {
		t.Errorf("compose = %q, want %q", comp, "local compose")
	}
}

func TestCompositeSource_FallsBackOnNotExist(t *testing.T) {
	local := &fakeTemplateSource{err: fs.ErrNotExist}
	embedded := &fakeTemplateSource{
		dockerfile: "FROM embedded",
		compose:    "embedded compose",
	}

	cs := &compositeSource{sources: []service.TemplateSource{local, embedded}}
	df, comp, err := cs.Load("base")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if df != "FROM embedded" {
		t.Errorf("dockerfile = %q, want %q", df, "FROM embedded")
	}
	if comp != "embedded compose" {
		t.Errorf("compose = %q, want %q", comp, "embedded compose")
	}
}

func TestCompositeSource_PropagatesRealErrors(t *testing.T) {
	local := &fakeTemplateSource{err: os.ErrPermission}
	embedded := &fakeTemplateSource{
		dockerfile: "FROM embedded",
		compose:    "embedded compose",
	}

	cs := &compositeSource{sources: []service.TemplateSource{local, embedded}}
	_, _, err := cs.Load("base")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("error = %v, want os.ErrPermission", err)
	}
}

func TestTemplateSourceForProject_IncludesLocalSource(t *testing.T) {
	dir := t.TempDir()

	// Create a local template.
	tmplDir := filepath.Join(dir, ".deckhand", "templates", "custom")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "Dockerfile.tmpl"), []byte("FROM custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "compose.yaml.tmpl"), []byte("services: {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := templateSourceForProject(dir)
	df, comp, err := src.Load("custom")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if df != "FROM custom" {
		t.Errorf("dockerfile = %q, want %q", df, "FROM custom")
	}
	if comp != "services: {}" {
		t.Errorf("compose = %q, want %q", comp, "services: {}")
	}
}

func TestLocalTemplateSource_ResolvesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	src := localTemplateSource(dir)

	expected := filepath.Join(dir, ".deckhand", "templates")
	if src.Dir != expected {
		t.Errorf("Dir = %q, want %q", src.Dir, expected)
	}
	if src.SourceLabel != "local" {
		t.Errorf("SourceLabel = %q, want %q", src.SourceLabel, "local")
	}
}
