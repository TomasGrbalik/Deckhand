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
	meta       *domain.TemplateMeta
	metaErr    error
}

func (f *fakeSource) Load(_ string) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.dockerfile, f.compose, nil
}

func (f *fakeSource) LoadMeta(_ string) (*domain.TemplateMeta, error) {
	if f.metaErr != nil {
		return nil, f.metaErr
	}
	if f.meta != nil {
		return f.meta, nil
	}
	return &domain.TemplateMeta{Name: "base", Description: "test template"}, nil
}

// Minimal templates that exercise the key template variables.
const fakeDockerfile = `FROM ubuntu:24.04
# Project: {{ .Name }}
`

const fakeCompose = `services:
  devcontainer:
    build:
      context: .
      dockerfile: Dockerfile
{{- if .Volumes }}
    volumes:
{{- range .Volumes }}
      - {{ .ComposeEntry }}
{{- end }}
{{- end }}
{{- if .ExposedPorts }}
    ports:
{{- range .ExposedPorts }}
      - "127.0.0.1:{{ .Port }}:{{ .Port }}"
{{- end }}
{{- end }}
{{- if .Environment }}
    environment:
{{- range .Environment }}
      {{ .Key }}: {{ printf "%q" .Value }}
{{- end }}
{{- end }}
    command: sleep infinity
{{- range .Companions }}

  {{ .Name }}:
    image: {{ .Image }}
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "{{ $.Name }}"
      dev.deckhand.service: "{{ .Name }}"
{{- if .Ports }}
    ports:
{{- range .Ports }}
      - "127.0.0.1:{{ . }}:{{ . }}"
{{- end }}
{{- end }}
{{- if .Environment }}
    environment:
{{- range .Environment }}
      {{ .Key }}: {{ printf "%q" .Value }}
{{- end }}
{{- end }}
{{- if .HealthCheck.Test }}
    healthcheck:
      test: ["CMD-SHELL", "{{ .HealthCheck.Test }}"]
      interval: {{ .HealthCheck.Interval }}
      timeout: {{ .HealthCheck.Timeout }}
      retries: {{ .HealthCheck.Retries }}
{{- end }}
{{- if .Volumes }}
    volumes:
{{- range .Volumes }}
      - {{ .ComposeEntry }}
{{- end }}
{{- end }}
{{- end }}
{{- if or .NamedVolumes .CompanionVolumes }}

volumes:
{{- range .NamedVolumes }}
  {{ .ComposeName }}:
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "{{ $.Name }}"
      dev.deckhand.volume: "{{ .MountName }}"
{{- end }}
{{- range .CompanionVolumes }}
  {{ .ComposeName }}:
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "{{ $.Name }}"
      dev.deckhand.service: "{{ .ServiceName }}"
{{- end }}
{{- end }}
`

func newFakeSource() *fakeSource {
	return &fakeSource{
		dockerfile: fakeDockerfile,
		compose:    fakeCompose,
	}
}

func TestRender_WithPorts(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Ports:    []domain.PortMapping{{Port: 8080}},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Dockerfile, "Project: myapp") {
		t.Errorf("Dockerfile missing project name\nGot:\n%s", out.Dockerfile)
	}

	if !strings.Contains(out.Compose, "127.0.0.1:8080:8080") {
		t.Errorf("compose missing port mapping\nGot:\n%s", out.Compose)
	}
}

func TestRender_NoPorts(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
	}

	out, err := svc.Render(project, domain.Mounts{})
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
	}, nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "", // empty — should default to "base"
	}

	_, err := svc.Render(project, domain.Mounts{})
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

func (c *nameCapture) LoadMeta(name string) (*domain.TemplateMeta, error) {
	return c.inner.LoadMeta(name)
}

func TestRender_TemplateNotFound(t *testing.T) {
	source := &fakeSource{
		err: errors.New("template not found"),
	}
	svc := service.NewTemplateService(source, nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "nonexistent",
	}

	_, err := svc.Render(project, domain.Mounts{})
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
	svc := service.NewTemplateService(source, nil)

	project := domain.Project{Name: "myapp", Template: "base"}

	_, err := svc.Render(project, domain.Mounts{})
	if err == nil {
		t.Fatal("expected error for malformed template, got nil")
	}
}

func TestRender_InternalPortsFiltered(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Ports: []domain.PortMapping{
			{Port: 8080, Internal: false},
			{Port: 5432, Internal: true}, // should be filtered out
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
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

func TestRender_VariableDefaults(t *testing.T) {
	source := &fakeSource{
		dockerfile: `FROM golang:{{ .Vars.go_version }}`,
		compose:    fakeCompose,
		meta: &domain.TemplateMeta{
			Name: "go",
			Variables: map[string]domain.TemplateVariable{
				"go_version": {Default: "1.23", Description: "Go version"},
			},
		},
	}
	svc := service.NewTemplateService(source, nil)

	project := domain.Project{Name: "myapp", Template: "go"}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Dockerfile, "golang:1.23") {
		t.Errorf("Dockerfile should use default go_version 1.23\nGot:\n%s", out.Dockerfile)
	}
}

func TestRender_VariableOverride(t *testing.T) {
	source := &fakeSource{
		dockerfile: `FROM golang:{{ .Vars.go_version }}`,
		compose:    fakeCompose,
		meta: &domain.TemplateMeta{
			Name: "go",
			Variables: map[string]domain.TemplateVariable{
				"go_version": {Default: "1.23", Description: "Go version"},
			},
		},
	}
	svc := service.NewTemplateService(source, nil)

	project := domain.Project{
		Name:      "myapp",
		Template:  "go",
		Variables: map[string]string{"go_version": "1.22"},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Dockerfile, "golang:1.22") {
		t.Errorf("Dockerfile should use overridden go_version 1.22\nGot:\n%s", out.Dockerfile)
	}
}

func TestRender_UnknownVariableIgnored(t *testing.T) {
	source := &fakeSource{
		dockerfile: `{{- if index .Vars "foo" -}}present{{- else -}}absent{{- end -}}`,
		compose:    fakeCompose,
		meta: &domain.TemplateMeta{
			Name:      "base",
			Variables: map[string]domain.TemplateVariable{},
		},
	}
	svc := service.NewTemplateService(source, nil)

	project := domain.Project{
		Name:      "myapp",
		Template:  "base",
		Variables: map[string]string{"foo": "bar"},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() should not error for unknown variables: %v", err)
	}

	if strings.Contains(out.Dockerfile, "present") {
		t.Errorf("unknown variable should not be available in Vars\nGot:\n%s", out.Dockerfile)
	}
}

func TestRender_NoVariablesField(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		// Variables not set — should default to empty map
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Dockerfile, "Project: myapp") {
		t.Errorf("Dockerfile should render normally\nGot:\n%s", out.Dockerfile)
	}
}

// --- Mount rendering tests ---

func TestRender_NamedVolume(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{Name: "myapp", Template: "base"}
	mounts := domain.Mounts{
		Volumes: []domain.VolumeMount{
			{Name: "workspace", Target: "/workspace"},
		},
	}

	out, err := svc.Render(project, mounts)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, "myapp-workspace:/workspace") {
		t.Errorf("compose missing named volume entry\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, "volumes:\n  myapp-workspace:") {
		t.Errorf("compose missing top-level volumes declaration\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, `dev.deckhand.managed: "true"`) {
		t.Errorf("compose missing deckhand managed label\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, `dev.deckhand.project: "myapp"`) {
		t.Errorf("compose missing deckhand project label\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, `dev.deckhand.volume: "workspace"`) {
		t.Errorf("compose missing deckhand volume label\nGot:\n%s", out.Compose)
	}
}

func TestRender_SecretFileBindMount(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{Name: "myapp", Template: "base"}
	mounts := domain.Mounts{
		Secrets: []domain.SecretMount{
			{
				Name:     "gitconfig",
				Source:   "/home/user/.gitconfig",
				Target:   "/home/dev/.gitconfig",
				ReadOnly: true,
			},
		},
	}

	out, err := svc.Render(project, mounts)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, "/home/user/.gitconfig:/home/dev/.gitconfig:ro") {
		t.Errorf("compose missing secret file bind mount\nGot:\n%s", out.Compose)
	}
}

func TestRender_SecretEnvVar(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{Name: "myapp", Template: "base"}
	mounts := domain.Mounts{
		Secrets: []domain.SecretMount{
			{
				Name:   "gh-token",
				Source: "ghp_abc",
				Env:    "GH_TOKEN",
			},
		},
	}

	out, err := svc.Render(project, mounts)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, `GH_TOKEN: "ghp_abc"`) {
		t.Errorf("compose missing secret env var\nGot:\n%s", out.Compose)
	}

	// Env-only secret should NOT produce a bind mount.
	for line := range strings.SplitSeq(out.Compose, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ghp_abc:") {
			t.Errorf("env-only secret should not produce a bind mount\nGot:\n%s", out.Compose)
		}
	}
}

func TestRender_SocketMount(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{Name: "myapp", Template: "base"}
	mounts := domain.Mounts{
		Sockets: []domain.SocketMount{
			{
				Name:   "ssh-agent",
				Source: "/run/user/1000/ssh.sock",
				Target: "/run/ssh.sock",
				Env:    "SSH_AUTH_SOCK",
			},
		},
	}

	out, err := svc.Render(project, mounts)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, "/run/user/1000/ssh.sock:/run/ssh.sock") {
		t.Errorf("compose missing socket bind mount\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, `SSH_AUTH_SOCK: "/run/ssh.sock"`) {
		t.Errorf("compose missing socket env var\nGot:\n%s", out.Compose)
	}
}

func TestRender_EnvKeyCollision(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Env:      map[string]string{"GH_TOKEN": "hardcoded"},
	}
	mounts := domain.Mounts{
		Secrets: []domain.SecretMount{
			{
				Name:   "gh-token",
				Source: "fromenv",
				Env:    "GH_TOKEN",
			},
		},
	}

	out, err := svc.Render(project, mounts)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, `GH_TOKEN: "fromenv"`) {
		t.Errorf("secret should override static env\nGot:\n%s", out.Compose)
	}

	if strings.Contains(out.Compose, "hardcoded") {
		t.Errorf("static env value should be overridden\nGot:\n%s", out.Compose)
	}
}

func TestRender_EnvironmentSorted(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Env: map[string]string{
			"Z_VAR": "z",
			"A_VAR": "a",
			"M_VAR": "m",
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	aIdx := strings.Index(out.Compose, "A_VAR")
	mIdx := strings.Index(out.Compose, "M_VAR")
	zIdx := strings.Index(out.Compose, "Z_VAR")

	if aIdx < 0 || mIdx < 0 || zIdx < 0 {
		t.Fatalf("compose missing env vars\nGot:\n%s", out.Compose)
	}

	if aIdx >= mIdx || mIdx >= zIdx {
		t.Errorf("environment vars not in alphabetical order\nGot:\n%s", out.Compose)
	}
}

func TestRender_ZeroMounts(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if strings.Contains(out.Compose, "volumes:") {
		t.Errorf("compose should not have volumes section with zero mounts\nGot:\n%s", out.Compose)
	}

	if strings.Contains(out.Compose, "environment:") {
		t.Errorf("compose should not have environment section with zero mounts and no env\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, "context: .") {
		t.Errorf("compose should have build context '.'\nGot:\n%s", out.Compose)
	}

	if !strings.Contains(out.Compose, "dockerfile: Dockerfile") {
		t.Errorf("compose should have dockerfile 'Dockerfile'\nGot:\n%s", out.Compose)
	}
}

// --- Companion rendering tests ---

func TestRender_WithCompanions(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "my-api",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: true},
			{Name: "redis", Enabled: true},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Postgres service block.
	if !strings.Contains(out.Compose, "\n  postgres:\n") {
		t.Errorf("compose missing postgres service block\nGot:\n%s", out.Compose)
	}
	if !strings.Contains(out.Compose, "image: postgres:16-alpine") {
		t.Errorf("compose missing postgres image\nGot:\n%s", out.Compose)
	}

	// Redis service block.
	if !strings.Contains(out.Compose, "\n  redis:\n") {
		t.Errorf("compose missing redis service block\nGot:\n%s", out.Compose)
	}
	if !strings.Contains(out.Compose, "image: redis:7-alpine") {
		t.Errorf("compose missing redis image\nGot:\n%s", out.Compose)
	}

	// Devcontainer still present.
	if !strings.Contains(out.Compose, "devcontainer:") {
		t.Errorf("compose missing devcontainer block\nGot:\n%s", out.Compose)
	}
}

func TestRender_CompanionLabels(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "my-api",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: true},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, `dev.deckhand.service: "postgres"`) {
		t.Errorf("compose missing deckhand service label for postgres\nGot:\n%s", out.Compose)
	}
	if !strings.Contains(out.Compose, `dev.deckhand.project: "my-api"`) {
		t.Errorf("compose missing deckhand project label\nGot:\n%s", out.Compose)
	}
}

func TestRender_CompanionPortsLocalhost(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: true},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, "127.0.0.1:5432:5432") {
		t.Errorf("compose missing localhost-bound port for postgres\nGot:\n%s", out.Compose)
	}
}

func TestRender_CompanionHealthcheck(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: true},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out.Compose, `["CMD-SHELL", "pg_isready -U dev"]`) {
		t.Errorf("compose missing healthcheck test for postgres\nGot:\n%s", out.Compose)
	}
	if !strings.Contains(out.Compose, "interval: 5s") {
		t.Errorf("compose missing healthcheck interval\nGot:\n%s", out.Compose)
	}
	if !strings.Contains(out.Compose, "timeout: 3s") {
		t.Errorf("compose missing healthcheck timeout\nGot:\n%s", out.Compose)
	}
	if !strings.Contains(out.Compose, "retries: 5") {
		t.Errorf("compose missing healthcheck retries\nGot:\n%s", out.Compose)
	}
}

func TestRender_CompanionVolumes(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: true},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Companion volume in service block.
	if !strings.Contains(out.Compose, "myapp-postgres-data:/var/lib/postgresql/data") {
		t.Errorf("compose missing postgres volume mount\nGot:\n%s", out.Compose)
	}

	// Top-level volume declaration.
	if !strings.Contains(out.Compose, "volumes:\n  myapp-postgres-data:") {
		t.Errorf("compose missing top-level postgres volume declaration\nGot:\n%s", out.Compose)
	}

	// Volume label uses service name.
	if !strings.Contains(out.Compose, `dev.deckhand.service: "postgres"`) {
		t.Errorf("compose missing service label on companion volume\nGot:\n%s", out.Compose)
	}
}

func TestRender_NoCompanionsBackwardCompat(t *testing.T) {
	svc := service.NewTemplateService(newFakeSource(), nil)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// No companion blocks should appear.
	if strings.Contains(out.Compose, "image:") {
		t.Errorf("compose should not have image: when no companions selected\nGot:\n%s", out.Compose)
	}
	if strings.Contains(out.Compose, "healthcheck:") {
		t.Errorf("compose should not have healthcheck when no companions selected\nGot:\n%s", out.Compose)
	}
	// Devcontainer should still be there.
	if !strings.Contains(out.Compose, "devcontainer:") {
		t.Errorf("compose missing devcontainer\nGot:\n%s", out.Compose)
	}
}

func TestRender_DisabledCompanionSkipped(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: false},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if strings.Contains(out.Compose, "postgres:") {
		t.Errorf("disabled companion should not appear in compose\nGot:\n%s", out.Compose)
	}
}

func TestRender_UnknownCompanionError(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "mysql", Enabled: true},
		},
	}

	_, err := svc.Render(project, domain.Mounts{})
	if err == nil {
		t.Fatal("expected error for unknown companion service, got nil")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("error should mention service name, got: %v", err)
	}
}

func TestRender_CompanionEnvironmentSorted(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc := service.NewTemplateService(newFakeSource(), reg)

	project := domain.Project{
		Name:     "myapp",
		Template: "base",
		Services: []domain.ServiceConfig{
			{Name: "postgres", Enabled: true},
		},
	}

	out, err := svc.Render(project, domain.Mounts{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Postgres env vars should be sorted: POSTGRES_DB < POSTGRES_PASSWORD < POSTGRES_USER.
	dbIdx := strings.Index(out.Compose, "POSTGRES_DB")
	pwIdx := strings.Index(out.Compose, "POSTGRES_PASSWORD")
	userIdx := strings.Index(out.Compose, "POSTGRES_USER")

	if dbIdx < 0 || pwIdx < 0 || userIdx < 0 {
		t.Fatalf("compose missing postgres env vars\nGot:\n%s", out.Compose)
	}

	if dbIdx >= pwIdx || pwIdx >= userIdx {
		t.Errorf("postgres env vars not in alphabetical order\nGot:\n%s", out.Compose)
	}
}
