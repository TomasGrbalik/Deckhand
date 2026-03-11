package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

// fakeSource is a test double for TemplateSource that returns canned templates.
type fakeSource struct {
	dockerfile string
	compose    string
	err        error
}

func (f *fakeSource) Load(_ string) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.dockerfile, f.compose, nil
}

// Minimal templates that exercise the key template variables.
const fakeDockerfile = `FROM ubuntu:24.04
# Project: {{ .Name }}
`

const fakeCompose = `services:
  devcontainer:
    build:
      context: ..
    volumes:
      - ..:/workspace
{{- if .ExposedPorts }}
    ports:
{{- range .ExposedPorts }}
      - "127.0.0.1:{{ .Port }}:{{ .Port }}"
{{- end }}
{{- end }}
    command: sleep infinity
`

func newFakeSource() *fakeSource {
	return &fakeSource{
		dockerfile: fakeDockerfile,
		compose:    fakeCompose,
	}
}

func TestRender_WithPorts(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource())

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Ports:    []domain.PortMapping{{Port: 8080}},
	}

	out, err := svc.Render(project)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Dockerfile, "Project: myapp") {
		t.Errorf("Dockerfile missing project name\nGot:\n%s", out.Dockerfile)
	}

	if !strings.Contains(out.Compose, "127.0.0.1:8080:8080") {
		t.Errorf("compose missing port mapping\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, "..:/workspace") {
		t.Errorf("compose missing workspace volume\nGot:\n%s", out.Compose)
	}
}

func TestRender_NoPorts(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource())

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
	}

	out, err := svc.Render(project)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if strings.Contains(out.Compose, "ports") {
		t.Errorf("compose should not have ports section\nGot:\n%s", out.Compose)
	}
}

func TestRender_DefaultTemplateName(t *testing.T) {
	// Track which name was requested.
	var loadedName string
	source := &fakeSource{
		dockerfile: fakeDockerfile,
		compose:    fakeCompose,
	}
	// Wrap the fake to capture the name.
	svc := service.NewTemplateService(&nameCapture{
		inner: source,
		name:  &loadedName,
	})

	project := domain.Project{
		Name:     "myapp",
		Template: "", // empty — should default to "base"
	}

	_, err := svc.Render(project)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if loadedName != "base" {
		t.Errorf("expected default template name %q, got %q", "base", loadedName)
	}
}

// nameCapture wraps a TemplateSource to record the name passed to Load.
type nameCapture struct {
	inner service.TemplateSource
	name  *string
}

func (c *nameCapture) Load(name string) (string, string, error) {
	*c.name = name
	return c.inner.Load(name)
}

func TestRender_TemplateNotFound(t *testing.T) {
	source := &fakeSource{
		err: errors.New("template not found"),
	}
	svc := service.NewTemplateService(source)

	project := domain.Project{
		Name:     "myapp",
		Template: "nonexistent",
	}

	_, err := svc.Render(project)
	if err == nil {
		t.Fatal("expected error for missing template, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention template name, got: %v", err)
	}
}

func TestRender_MalformedTemplate(t *testing.T) {
	source := &fakeSource{
		dockerfile: "{{ .Invalid | bad_func }}",
		compose:    fakeCompose,
	}
	svc := service.NewTemplateService(source)

	project := domain.Project{Name: "myapp", Template: "base"}

	_, err := svc.Render(project)
	if err == nil {
		t.Fatal("expected error for malformed template, got nil")
	}
}

func TestRender_InternalPortsFiltered(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource())

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Ports: []domain.PortMapping{
			{Port: 8080, Internal: false},
			{Port: 5432, Internal: true}, // should be filtered out
		},
	}

	out, err := svc.Render(project)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, "127.0.0.1:8080:8080") {
		t.Errorf("compose missing exposed port 8080\nGot:\n%s", out.Compose)
	}

	if strings.Contains(out.Compose, "5432") {
		t.Errorf("compose should not contain internal port 5432\nGot:\n%s", out.Compose)
	}
}
