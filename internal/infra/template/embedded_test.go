package template_test

import (
	"strings"
	"testing"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	tmpl "github.com/TomasGrbalik/deckhand/internal/infra/template"
)

func TestLoadMeta_BaseTemplate(t *testing.T) {
	meta, err := tmpl.LoadMeta("base")
	if err != nil {
		t.Fatalf("LoadMeta(\"base\") returned error: %v", err)
	}

	if meta.Name != "base" {
		t.Errorf("Name = %q, want %q", meta.Name, "base")
	}
	if meta.Description == "" {
		t.Error("Description should not be empty")
	}
	if meta.Variables == nil {
		t.Error("Variables should not be nil (expected empty map)")
	}
}

func TestLoadMeta_BaseTemplateDeclaresWorkspaceVolume(t *testing.T) {
	meta, err := tmpl.LoadMeta("base")
	if err != nil {
		t.Fatalf("LoadMeta(\"base\") returned error: %v", err)
	}

	if len(meta.Mounts.Volumes) == 0 {
		t.Fatal("base template should declare at least one volume mount")
	}
	found := false
	for _, v := range meta.Mounts.Volumes {
		if v.Name == "workspace" && v.Target == "/workspace" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("base template should declare workspace volume with target /workspace, got %+v", meta.Mounts.Volumes)
	}
}

func TestLoadMeta_PythonTemplateDeclaresWorkspaceVolume(t *testing.T) {
	meta, err := tmpl.LoadMeta("python")
	if err != nil {
		t.Fatalf("LoadMeta(\"python\") returned error: %v", err)
	}

	if len(meta.Mounts.Volumes) == 0 {
		t.Fatal("python template should declare at least one volume mount")
	}
	found := false
	for _, v := range meta.Mounts.Volumes {
		if v.Name == "workspace" && v.Target == "/workspace" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("python template should declare workspace volume with target /workspace, got %+v", meta.Mounts.Volumes)
	}
}

func TestLoadMeta_NonexistentTemplate(t *testing.T) {
	_, err := tmpl.LoadMeta("nonexistent")
	if err == nil {
		t.Fatal("LoadMeta(\"nonexistent\") should return an error")
	}
}

func TestLoad_BaseTemplate(t *testing.T) {
	dockerfile, compose, err := tmpl.Load("base")
	if err != nil {
		t.Fatalf("Load(\"base\") returned error: %v", err)
	}

	if dockerfile == "" {
		t.Error("Dockerfile template is empty")
	}
	if compose == "" {
		t.Error("compose template is empty")
	}
}

func TestLoad_NonexistentTemplate(t *testing.T) {
	_, _, err := tmpl.Load("nonexistent")
	if err == nil {
		t.Fatal("Load(\"nonexistent\") should return an error")
	}
}

func TestLoad_DockerfileContent(t *testing.T) {
	dockerfile, _, err := tmpl.Load("base")
	if err != nil {
		t.Fatalf("Load(\"base\") returned error: %v", err)
	}

	checks := []string{
		"DO NOT EDIT",
		"Source: .deckhand.yaml",
		"Regenerate with: deckhand up",
		"ubuntu:24.04",
		"git",
		"curl",
		"wget",
		"ripgrep",
		"build-essential",
		"zsh",
		"1000",
		"/workspace",
		"USER dev",
	}

	for _, want := range checks {
		if !strings.Contains(dockerfile, want) {
			t.Errorf("Dockerfile missing %q", want)
		}
	}
}

func TestLoadMeta_PythonTemplate(t *testing.T) {
	meta, err := tmpl.LoadMeta("python")
	if err != nil {
		t.Fatalf("LoadMeta(\"python\") returned error: %v", err)
	}

	if meta.Name != "python" {
		t.Errorf("Name = %q, want %q", meta.Name, "python")
	}
	if meta.Description == "" {
		t.Error("Description should not be empty")
	}
	v, ok := meta.Variables["python_version"]
	if !ok {
		t.Fatal("Variables missing \"python_version\"")
	}
	if v.Default != "3.12" {
		t.Errorf("python_version default = %q, want %q", v.Default, "3.12")
	}
}

func TestLoad_PythonTemplate(t *testing.T) {
	dockerfile, compose, err := tmpl.Load("python")
	if err != nil {
		t.Fatalf("Load(\"python\") returned error: %v", err)
	}

	if dockerfile == "" {
		t.Error("Dockerfile template is empty")
	}
	if compose == "" {
		t.Error("compose template is empty")
	}
}

func TestLoad_PythonDockerfileContent(t *testing.T) {
	dockerfile, _, err := tmpl.Load("python")
	if err != nil {
		t.Fatalf("Load(\"python\") returned error: %v", err)
	}

	checks := []string{
		"DO NOT EDIT",
		"Source: .deckhand.yaml",
		"Regenerate with: deckhand up",
		"python:{{ .Vars.python_version }}-slim",
		"pyright",
		"debugpy",
		"git",
		"/workspace",
		"USER dev",
	}

	for _, want := range checks {
		if !strings.Contains(dockerfile, want) {
			t.Errorf("Python Dockerfile missing %q", want)
		}
	}
}

func TestRender_PythonDockerfileDefaults(t *testing.T) {
	dockerfile, _, err := tmpl.Load("python")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	parsed, err := template.New("dockerfile").Parse(dockerfile)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	data := struct {
		Vars map[string]string
	}{
		Vars: map[string]string{"python_version": "3.12"},
	}

	var buf strings.Builder
	if err := parsed.Execute(&buf, data); err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "python:3.12-slim") {
		t.Errorf("rendered Dockerfile missing python:3.12-slim\nGot:\n%s", output)
	}
}

func TestRender_PythonDockerfileCustomVersion(t *testing.T) {
	dockerfile, _, err := tmpl.Load("python")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	parsed, err := template.New("dockerfile").Parse(dockerfile)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	data := struct {
		Vars map[string]string
	}{
		Vars: map[string]string{"python_version": "3.11"},
	}

	var buf strings.Builder
	if err := parsed.Execute(&buf, data); err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "python:3.11-slim") {
		t.Errorf("rendered Dockerfile missing python:3.11-slim\nGot:\n%s", output)
	}
	if strings.Contains(output, "3.12") {
		t.Errorf("rendered Dockerfile should not contain default version 3.12 when overridden\nGot:\n%s", output)
	}
}

func TestRender_PythonComposeWithPorts(t *testing.T) {
	output := renderCompose(t, "python", templateData{
		Project: domain.Project{
			Name:     "myapp",
			Template: "python",
			Ports: []domain.PortMapping{
				{Port: 8000, Name: "web"},
			},
		},
		ExposedPorts: []domain.PortMapping{
			{Port: 8000, Name: "web"},
		},
	})

	checks := []string{
		"DO NOT EDIT",
		"dev.deckhand.managed",
		`dev.deckhand.project: "myapp"`,
		"context: .",
		"dockerfile: Dockerfile",
		"127.0.0.1:8000:8000",
		"sleep infinity",
	}

	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("python compose output missing %q\nGot:\n%s", want, output)
		}
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("rendered compose is not valid YAML: %v\nGot:\n%s", err, output)
	}
}

// VolumeEntry mirrors the service layer's VolumeEntry for template rendering.
type VolumeEntry struct {
	entry string
}

// ComposeEntry returns the compose-format volume string.
func (v VolumeEntry) ComposeEntry() string {
	return v.entry
}

// EnvEntry mirrors the service layer's EnvEntry for template rendering.
type EnvEntry struct {
	Key   string
	Value string
}

// NamedVolumeEntry mirrors the service layer's NamedVolumeEntry.
type NamedVolumeEntry struct {
	ComposeName string
	MountName   string
}

// CompanionTemplateData mirrors the service layer's CompanionTemplateData.
type CompanionTemplateData struct {
	Name        string
	Image       string
	Ports       []int
	Environment []EnvEntry
	HealthCheck domain.HealthCheck
	Volumes     []VolumeEntry
}

// CompanionVolumeEntry mirrors the service layer's CompanionVolumeEntry.
type CompanionVolumeEntry struct {
	ComposeName string
	ServiceName string
}

// templateData mirrors the data passed to compose templates during rendering.
// ExposedPorts contains only non-internal ports from domain.Project.
type templateData struct {
	domain.Project
	ExposedPorts     []domain.PortMapping
	Volumes          []VolumeEntry
	Environment      []EnvEntry
	NamedVolumes     []NamedVolumeEntry
	Companions       []CompanionTemplateData
	CompanionVolumes []CompanionVolumeEntry
	NetworkName      string
	NetworkIP        string
	Command          string
}

// renderCompose is a test helper that loads, parses, and executes a
// compose template with the given data.
func renderCompose(t *testing.T, templateName string, data templateData) string {
	t.Helper()

	// The compose template always reads .Command; the service layer is
	// responsible for defaulting it. In these tests we mimic that default
	// so callers don't have to set it explicitly.
	if data.Command == "" {
		data.Command = "sleep infinity"
	}

	_, composeTmpl, err := tmpl.Load(templateName)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	parsed, err := template.New("compose").Parse(composeTmpl)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	var buf strings.Builder
	if err := parsed.Execute(&buf, data); err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	return buf.String()
}

func TestRender_ComposeWithPorts(t *testing.T) {
	output := renderCompose(t, "base", templateData{
		Project: domain.Project{
			Name:     "myapp",
			Template: "base",
			Ports: []domain.PortMapping{
				{Port: 8080, Name: "web"},
				{Port: 3000, Name: "frontend"},
			},
		},
		ExposedPorts: []domain.PortMapping{
			{Port: 8080, Name: "web"},
			{Port: 3000, Name: "frontend"},
		},
	})

	checks := []string{
		"DO NOT EDIT",
		"Source: .deckhand.yaml",
		"Regenerate with: deckhand up",
		"devcontainer",
		"context: .",
		"dockerfile: Dockerfile",
		"127.0.0.1:8080:8080",
		"127.0.0.1:3000:3000",
		"sleep infinity",
	}

	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("compose output missing %q\nGot:\n%s", want, output)
		}
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("rendered compose is not valid YAML: %v\nGot:\n%s", err, output)
	}
}

func TestRender_ComposeWithNoPorts(t *testing.T) {
	output := renderCompose(t, "base", templateData{
		Project: domain.Project{
			Name:     "myapp",
			Template: "base",
		},
	})

	if strings.Contains(output, "ports") {
		t.Errorf("compose output should not contain ports section when no ports defined\nGot:\n%s", output)
	}

	if !strings.Contains(output, "sleep infinity") {
		t.Errorf("compose output missing 'sleep infinity'\nGot:\n%s", output)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("rendered compose is not valid YAML: %v\nGot:\n%s", err, output)
	}
}

func TestRender_ComposeWithCompanions(t *testing.T) {
	templates := []string{"base", "python"}

	for _, tmplName := range templates {
		t.Run(tmplName, func(t *testing.T) {
			output := renderCompose(t, tmplName, templateData{
				Project: domain.Project{
					Name:     "myapp",
					Template: tmplName,
					Ports: []domain.PortMapping{
						{Port: 8080, Name: "web"},
					},
				},
				ExposedPorts: []domain.PortMapping{
					{Port: 8080, Name: "web"},
				},
				Companions: []CompanionTemplateData{
					{
						Name:  "postgres",
						Image: "postgres:16-alpine",
						Ports: []int{5432},
						Environment: []EnvEntry{
							{Key: "POSTGRES_DB", Value: "devdb"},
							{Key: "POSTGRES_PASSWORD", Value: "dev"},
							{Key: "POSTGRES_USER", Value: "dev"},
						},
						HealthCheck: domain.HealthCheck{
							Test:     "pg_isready -U dev",
							Interval: "5s",
							Timeout:  "3s",
							Retries:  5,
						},
						Volumes: []VolumeEntry{{entry: "myapp-postgres-data:/var/lib/postgresql/data"}},
					},
					{
						Name:  "redis",
						Image: "redis:7-alpine",
						Ports: []int{6379},
						HealthCheck: domain.HealthCheck{
							Test:     "redis-cli ping",
							Interval: "5s",
							Timeout:  "3s",
							Retries:  5,
						},
						Volumes: []VolumeEntry{{entry: "myapp-redis-data:/data"}},
					},
				},
				CompanionVolumes: []CompanionVolumeEntry{
					{ComposeName: "myapp-postgres-data", ServiceName: "postgres"},
					{ComposeName: "myapp-redis-data", ServiceName: "redis"},
				},
			})

			// Must be valid YAML.
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
				t.Fatalf("rendered compose is not valid YAML: %v\nGot:\n%s", err, output)
			}

			// Services map must contain companion keys.
			services, ok := parsed["services"].(map[string]any)
			if !ok {
				t.Fatalf("expected services map in YAML\nGot:\n%s", output)
			}
			for _, name := range []string{"postgres", "redis", "devcontainer"} {
				if _, found := services[name]; !found {
					t.Errorf("services map missing %q\nGot:\n%s", name, output)
				}
			}

			// Companion content checks.
			checks := []string{
				"image: postgres:16-alpine",
				"image: redis:7-alpine",
				"127.0.0.1:5432:5432",
				"127.0.0.1:6379:6379",
				"POSTGRES_DB:",
				"POSTGRES_USER:",
				"pg_isready",
				"redis-cli ping",
				"myapp-postgres-data:",
				"myapp-redis-data:",
			}
			for _, want := range checks {
				if !strings.Contains(output, want) {
					t.Errorf("compose output missing %q\nGot:\n%s", want, output)
				}
			}

			// Top-level volumes section must declare companion volumes.
			vols, ok := parsed["volumes"].(map[string]any)
			if !ok {
				t.Fatalf("expected top-level volumes map\nGot:\n%s", output)
			}
			for _, volName := range []string{"myapp-postgres-data", "myapp-redis-data"} {
				if _, ok := vols[volName]; !ok {
					t.Errorf("volumes map missing %q\nGot:\n%s", volName, output)
				}
			}
		})
	}
}
