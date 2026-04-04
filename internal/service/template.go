package service

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
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

// CompanionResolver looks up companion service definitions by name.
// The CompanionRegistry satisfies this interface.
type CompanionResolver interface {
	Resolve(name, version string) (domain.CompanionService, error)
}

// CompanionTemplateData holds the template-friendly representation of a
// companion service for rendering into compose YAML.
type CompanionTemplateData struct {
	Name        string
	Image       string
	Ports       []int
	Environment []EnvEntry
	HealthCheck domain.HealthCheck
	Volumes     []VolumeEntry
}

// CompanionVolumeEntry represents a companion's named volume in the top-level
// volumes: section, labeled with the owning service name.
type CompanionVolumeEntry struct {
	ComposeName string
	ServiceName string
}

// templateData is the data structure passed to Go templates during rendering.
// ExposedPorts contains only non-internal ports from the project config.
// Vars contains template variables with defaults merged with project overrides.
type templateData struct {
	domain.Project
	ExposedPorts     []domain.PortMapping
	Vars             map[string]string
	Volumes          []VolumeEntry
	Environment      []EnvEntry
	NamedVolumes     []NamedVolumeEntry
	Companions       []CompanionTemplateData
	CompanionVolumes []CompanionVolumeEntry
}

// TemplateService renders project templates into Dockerfile and compose content.
type TemplateService struct {
	source   TemplateSource
	registry CompanionResolver
}

// NewTemplateService creates a TemplateService with the given template source
// and an optional companion registry. Pass nil if companion services are not
// needed (e.g., when no services are selected in the project config).
func NewTemplateService(source TemplateSource, registry CompanionResolver) *TemplateService {
	return &TemplateService{source: source, registry: registry}
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

	data, err := buildTemplateData(project, meta, mounts, s.registry)
	if err != nil {
		return nil, fmt.Errorf("building template data: %w", err)
	}

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
func buildTemplateData(project domain.Project, meta *domain.TemplateMeta, mounts domain.Mounts, registry CompanionResolver) (templateData, error) {
	var exposed []domain.PortMapping
	for _, p := range project.Ports {
		if !p.Internal {
			exposed = append(exposed, p)
		}
	}

	vars := mergeVars(meta, project.Variables)
	volumes, namedVolumes := buildVolumeEntries(project.Name, mounts)
	environment := buildEnvironment(project.Env, mounts)

	companions, companionVolumes, err := buildCompanionData(project, registry)
	if err != nil {
		return templateData{}, err
	}

	return templateData{
		Project:          project,
		ExposedPorts:     exposed,
		Vars:             vars,
		Volumes:          volumes,
		Environment:      environment,
		NamedVolumes:     namedVolumes,
		Companions:       companions,
		CompanionVolumes: companionVolumes,
	}, nil
}

// buildCompanionData resolves the project's selected services via the registry
// and converts them into template-friendly data. Returns nil slices when
// registry is nil or no services are selected.
func buildCompanionData(project domain.Project, registry CompanionResolver) ([]CompanionTemplateData, []CompanionVolumeEntry, error) {
	if len(project.Services) == 0 {
		return nil, nil, nil
	}

	var companions []CompanionTemplateData
	var volumes []CompanionVolumeEntry

	for _, sc := range project.Services {
		if !sc.Enabled {
			continue
		}
		if registry == nil {
			return nil, nil, errors.New("companion registry is required when services are configured")
		}

		svc, err := registry.Resolve(sc.Name, sc.Version)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving companion %q: %w", sc.Name, err)
		}

		// Build sorted environment entries for deterministic output.
		envEntries := buildSortedEnv(svc.Environment)

		// Build volume entries and collect named volumes.
		var volEntries []VolumeEntry
		for _, v := range svc.Volumes {
			// Volumes are in "name:/path" format. Prefix name with project.
			name, target, ok := parseVolumeSpec(v)
			if !ok {
				return nil, nil, fmt.Errorf("companion %q has invalid volume spec %q", svc.Name, v)
			}
			composeName := project.Name + "-" + name
			volEntries = append(volEntries, VolumeEntry{entry: composeName + ":" + target})
			volumes = append(volumes, CompanionVolumeEntry{
				ComposeName: composeName,
				ServiceName: svc.Name,
			})
		}

		companions = append(companions, CompanionTemplateData{
			Name:        svc.Name,
			Image:       svc.Image,
			Ports:       svc.Ports,
			Environment: envEntries,
			HealthCheck: svc.HealthCheck,
			Volumes:     volEntries,
		})
	}

	return companions, volumes, nil
}

// buildSortedEnv converts a map of environment variables into a sorted slice
// of EnvEntry for deterministic template output.
func buildSortedEnv(env map[string]string) []EnvEntry {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]EnvEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, EnvEntry{Key: k, Value: env[k]})
	}
	return entries
}

// parseVolumeSpec splits a "name:/path" volume spec into name and target.
func parseVolumeSpec(spec string) (name, target string, ok bool) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
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
