package service

import (
	"bytes"
	"fmt"
	"maps"
	"sort"
	"text/template"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// TemplateSource loads raw template strings by name. Phase 1 uses embedded
// templates; later phases can swap in a git-based loader.
type TemplateSource interface {
	Load(name string) (dockerfile string, compose string, err error)
	LoadMeta(name string) (*domain.TemplateMeta, error)
}

// RenderedOutput holds the rendered Dockerfile and compose file content.
type RenderedOutput struct {
	Dockerfile string
	Compose    string
}

// VolumeEntry represents a single line in the compose volumes: list.
// It can be a named volume (e.g., "myapp-workspace:/workspace") or a bind
// mount (e.g., "/host/path:/container/path:ro").
type VolumeEntry struct {
	entry string
}

// ComposeEntry returns the compose-format volume string.
func (v VolumeEntry) ComposeEntry() string {
	return v.entry
}

// EnvEntry represents a single environment variable in compose output.
type EnvEntry struct {
	Key   string
	Value string
}

// NamedVolumeEntry represents a named volume that needs a top-level volumes:
// declaration with deckhand labels.
type NamedVolumeEntry struct {
	ComposeName string
	MountName   string
}

// templateData is the data structure passed to Go templates during rendering.
// ExposedPorts contains only non-internal ports from the project config.
// Vars contains template variables with defaults merged with project overrides.
type templateData struct {
	domain.Project
	ExposedPorts []domain.PortMapping
	Vars         map[string]string
	Volumes      []VolumeEntry
	Environment  []EnvEntry
	NamedVolumes []NamedVolumeEntry
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
// configuration and resolved mounts. If the project has no template set, it
// defaults to "base". Template variable defaults are merged with project
// overrides — project values take precedence.
func (s *TemplateService) Render(project domain.Project, mounts domain.Mounts) (*RenderedOutput, error) {
	name := project.Template
	if name == "" {
		name = "base"
	}

	dockerfileTmpl, composeTmpl, err := s.source.Load(name)
	if err != nil {
		return nil, fmt.Errorf("loading template %q: %w", name, err)
	}

	meta, err := s.source.LoadMeta(name)
	if err != nil {
		return nil, fmt.Errorf("loading metadata for template %q: %w", name, err)
	}

	data := buildTemplateData(project, meta, mounts)

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
// internal ports, merging template variable defaults with project overrides,
// and converting resolved mounts into template-friendly entries.
func buildTemplateData(project domain.Project, meta *domain.TemplateMeta, mounts domain.Mounts) templateData {
	var exposed []domain.PortMapping
	for _, p := range project.Ports {
		if !p.Internal {
			exposed = append(exposed, p)
		}
	}

	vars := mergeVars(meta, project.Variables)
	volumes, namedVolumes := buildVolumeEntries(project.Name, mounts)
	environment := buildEnvironment(project.Env, mounts)

	return templateData{
		Project:      project,
		ExposedPorts: exposed,
		Vars:         vars,
		Volumes:      volumes,
		Environment:  environment,
		NamedVolumes: namedVolumes,
	}
}

// buildVolumeEntries produces the compose volumes: entries and the top-level
// named volume declarations from resolved mounts.
func buildVolumeEntries(projectName string, mounts domain.Mounts) ([]VolumeEntry, []NamedVolumeEntry) {
	var volumes []VolumeEntry
	var named []NamedVolumeEntry

	// Named volumes from VolumeMount.
	for _, v := range mounts.Volumes {
		composeName := projectName + "-" + v.Name
		volumes = append(volumes, VolumeEntry{
			entry: composeName + ":" + v.Target,
		})
		named = append(named, NamedVolumeEntry{
			ComposeName: composeName,
			MountName:   v.Name,
		})
	}

	// Bind mounts from file-based secrets (those with Source + Target).
	for _, s := range mounts.Secrets {
		if s.Source == "" || s.Target == "" {
			continue
		}
		entry := s.Source + ":" + s.Target
		if s.ReadOnly {
			entry += ":ro"
		}
		volumes = append(volumes, VolumeEntry{entry: entry})
	}

	// Bind mounts from sockets (those with Source + Target).
	for _, s := range mounts.Sockets {
		if s.Source == "" || s.Target == "" {
			continue
		}
		volumes = append(volumes, VolumeEntry{
			entry: s.Source + ":" + s.Target,
		})
	}

	return volumes, named
}

// buildEnvironment merges static env vars from the project config with env
// vars injected by secrets and sockets. On key collision, secrets override
// static env. The result is sorted by key for deterministic output.
func buildEnvironment(staticEnv map[string]string, mounts domain.Mounts) []EnvEntry {
	merged := make(map[string]string)

	// Start with static env from project config.
	maps.Copy(merged, staticEnv)

	// Overlay secret env vars (secret wins on collision).
	for _, s := range mounts.Secrets {
		if s.Env == "" {
			continue
		}
		// Env-only secret: resolved source value becomes the env value.
		// File+env secret: target path becomes the env value.
		if s.Target != "" {
			merged[s.Env] = s.Target
		} else if s.Source != "" {
			merged[s.Env] = s.Source
		}
	}

	// Overlay socket env vars (target path becomes the env value).
	for _, s := range mounts.Sockets {
		if s.Env == "" || s.Target == "" {
			continue
		}
		merged[s.Env] = s.Target
	}

	if len(merged) == 0 {
		return nil
	}

	// Sort by key for deterministic output.
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]EnvEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, EnvEntry{Key: k, Value: merged[k]})
	}
	return entries
}

// mergeVars builds the final variable map: start with template defaults, then
// overlay project overrides. Unknown project variables are silently ignored.
func mergeVars(meta *domain.TemplateMeta, projectVars map[string]string) map[string]string {
	vars := make(map[string]string)

	// Start with template defaults.
	if meta != nil {
		for k, v := range meta.Variables {
			vars[k] = v.Default
		}
	}

	// Overlay project overrides (only for variables the template declares).
	for k, v := range projectVars {
		if meta != nil {
			if _, declared := meta.Variables[k]; declared {
				vars[k] = v
			}
		}
	}

	return vars
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
