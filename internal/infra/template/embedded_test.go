package template_test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	tmpl "github.com/TomasGrbalik/deckhand/internal/infra/template"
)

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
		"ubuntu:24.04",
		"git",
		"curl",
		"build-essential",
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

// templateData mirrors the data passed to compose templates during rendering.
// ExposedPorts contains only non-internal ports from domain.Project.
type templateData struct {
	domain.Project
	ExposedPorts []domain.PortMapping
}

func TestRender_ComposeWithPorts(t *testing.T) {
	_, composeTmpl, err := tmpl.Load("base")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	parsed, err := template.New("compose").Parse(composeTmpl)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	data := templateData{
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
	}

	var buf strings.Builder
	if err := parsed.Execute(&buf, data); err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	output := buf.String()

	checks := []string{
		"DO NOT EDIT",
		"devcontainer",
		".:/workspace",
		"127.0.0.1:8080:8080",
		"127.0.0.1:3000:3000",
		"sleep infinity",
	}

	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("compose output missing %q\nGot:\n%s", want, output)
		}
	}
}

func TestRender_ComposeWithNoPorts(t *testing.T) {
	_, composeTmpl, err := tmpl.Load("base")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	parsed, err := template.New("compose").Parse(composeTmpl)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	data := templateData{
		Project: domain.Project{
			Name:     "myapp",
			Template: "base",
		},
	}

	var buf strings.Builder
	if err := parsed.Execute(&buf, data); err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	output := buf.String()

	if strings.Contains(output, "ports") {
		t.Errorf("compose output should not contain ports section when no ports defined\nGot:\n%s", output)
	}

	if !strings.Contains(output, "sleep infinity") {
		t.Errorf("compose output missing 'sleep infinity'\nGot:\n%s", output)
	}
}
