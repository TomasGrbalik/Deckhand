package template

import (
	"fmt"
	"io/fs"
	"log"

	"gopkg.in/yaml.v3"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/templates"
)

// EmbeddedSource loads templates from the embedded filesystem.
// It implements service.TemplateSource.
type EmbeddedSource struct{}

// Load reads the raw Dockerfile and compose template strings for the given
// template name from the embedded filesystem.
func (e *EmbeddedSource) Load(name string) (dockerfile string, compose string, err error) {
	return Load(name)
}

// LoadMeta reads and parses the metadata.yaml for the given template name
// from the embedded filesystem.
func (e *EmbeddedSource) LoadMeta(name string) (*domain.TemplateMeta, error) {
	return LoadMeta(name)
}

// Load reads the raw Dockerfile and compose template strings for the given
// template name from the embedded filesystem. It does not render them —
// rendering is the responsibility of the service layer (#5).
//
// The compose template is resolved with a fallback: if the template directory
// contains its own compose.yaml.tmpl that file is used; otherwise the shared
// compose.yaml.tmpl at the root of the embedded filesystem is used.
func Load(name string) (dockerfile string, compose string, err error) {
	base := name

	df, err := templates.FS.ReadFile(base + "/Dockerfile.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("loading Dockerfile template %q: %w", name, err)
	}

	cf, err := templates.FS.ReadFile(base + "/compose.yaml.tmpl")
	if err != nil {
		// Fall back to the shared compose template.
		cf, err = templates.FS.ReadFile("compose.yaml.tmpl")
		if err != nil {
			return "", "", fmt.Errorf("loading compose template %q: %w", name, err)
		}
	}

	return string(df), string(cf), nil
}

// LoadMeta reads and parses metadata.yaml for the given template name
// from the embedded filesystem.
func LoadMeta(name string) (*domain.TemplateMeta, error) {
	data, err := templates.FS.ReadFile(name + "/metadata.yaml")
	if err != nil {
		return nil, fmt.Errorf("loading metadata for template %q: %w", name, err)
	}

	var meta domain.TemplateMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing metadata for template %q: %w", name, err)
	}

	return &meta, nil
}

// List returns TemplateInfo for every embedded template that has a valid
// metadata.yaml. Templates with missing or unparseable metadata are skipped
// with a log warning.
func (e *EmbeddedSource) List() ([]domain.TemplateInfo, error) {
	entries, err := fs.ReadDir(templates.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded templates: %w", err)
	}

	var result []domain.TemplateInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := LoadMeta(entry.Name())
		if err != nil {
			log.Printf("warning: skipping embedded template %q: %v", entry.Name(), err)
			continue
		}
		result = append(result, domain.TemplateInfo{
			Name:        meta.Name,
			Description: meta.Description,
			Source:      "builtin",
		})
	}
	return result, nil
}
