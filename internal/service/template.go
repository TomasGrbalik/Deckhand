package service

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// TemplateSource loads raw template strings by name. Phase 1 uses embedded
// templates; later phases can swap in a git-based loader.
type TemplateSource interface {
	Load(name string) (dockerfile string, compose string, err error)
}

// RenderedOutput holds the rendered Dockerfile and compose file content.
type RenderedOutput struct {
	Dockerfile string
	Compose    string
}

// templateData is the data structure passed to Go templates during rendering.
// ExposedPorts contains only non-internal ports from the project config.
type templateData struct {
	domain.Project
	ExposedPorts []domain.PortMapping
}

// TemplateService renders project templates into Dockerfile and compose content.
type TemplateService struct {
	source TemplateSource
}

// NewTemplateService creates a TemplateService with the given template source.
func NewTemplateService(source TemplateSource) *TemplateService {
	return &TemplateService{source: source}
}

// Render loads the template for the project and renders it with the project's
// configuration. If the project has no template set, it defaults to "base".
func (s *TemplateService) Render(project domain.Project) (*RenderedOutput, error) {
	name := project.Template
	if name == "" {
		name = "base"
	}

	dockerfileTmpl, composeTmpl, err := s.source.Load(name)
	if err != nil {
		return nil, fmt.Errorf("loading template %q: %w", name, err)
	}

	data := buildTemplateData(project)

	dockerfile, err := render("Dockerfile", dockerfileTmpl, data)
	if err != nil {
		return nil, fmt.Errorf("rendering Dockerfile: %w", err)
	}

	compose, err := render("compose", composeTmpl, data)
	if err != nil {
		return nil, fmt.Errorf("rendering compose: %w", err)
	}

	return &RenderedOutput{
		Dockerfile: dockerfile,
		Compose:    compose,
	}, nil
}

// buildTemplateData creates the data passed to templates, filtering out
// internal ports that should not be exposed on the host.
func buildTemplateData(project domain.Project) templateData {
	var exposed []domain.PortMapping
	for _, p := range project.Ports {
		if !p.Internal {
			exposed = append(exposed, p)
		}
	}
	return templateData{
		Project:      project,
		ExposedPorts: exposed,
	}
}

// render parses and executes a single Go template.
func render(name, tmplStr string, data templateData) (string, error) {
	t, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
