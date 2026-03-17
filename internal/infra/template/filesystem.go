package template

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// FilesystemSource loads templates from a directory on disk (e.g.
// ~/.config/deckhand/templates/). It implements the same listing interface as
// EmbeddedSource but reads from the real filesystem, allowing user-provided
// and overridden templates.
type FilesystemSource struct {
	Dir string // root directory containing template subdirectories
}

// Load reads the raw Dockerfile and compose template strings for the given
// template name from the filesystem directory.
func (f *FilesystemSource) Load(name string) (dockerfile string, compose string, err error) {
	base, err := f.templateDir(name)
	if err != nil {
		return "", "", err
	}

	df, err := os.ReadFile(filepath.Join(base, "Dockerfile.tmpl"))
	if err != nil {
		return "", "", fmt.Errorf("loading Dockerfile template %q: %w", name, err)
	}

	cf, err := os.ReadFile(filepath.Join(base, "compose.yaml.tmpl"))
	if err != nil {
		return "", "", fmt.Errorf("loading compose template %q: %w", name, err)
	}

	return string(df), string(cf), nil
}

// LoadMeta reads and parses metadata.yaml for the given template name from the
// filesystem directory.
func (f *FilesystemSource) LoadMeta(name string) (*domain.TemplateMeta, error) {
	base, err := f.templateDir(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(base, "metadata.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading metadata for template %q: %w", name, err)
	}

	var meta domain.TemplateMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing metadata for template %q: %w", name, err)
	}

	return &meta, nil
}

// templateDir validates a template name and returns the full directory path.
// It rejects names containing path separators or traversal components.
func (f *FilesystemSource) templateDir(name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || clean != filepath.Base(clean) || strings.ContainsRune(clean, os.PathSeparator) {
		return "", fmt.Errorf("invalid template name %q", name)
	}
	return filepath.Join(f.Dir, clean), nil
}

// List returns TemplateInfo for every template in the directory that has a
// valid metadata.yaml. If the directory does not exist, it returns an empty
// slice (not an error). Templates with missing or bad metadata are skipped
// with a log warning.
func (f *FilesystemSource) List() ([]domain.TemplateInfo, error) {
	entries, err := os.ReadDir(f.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading template directory %q: %w", f.Dir, err)
	}

	var result []domain.TemplateInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := f.LoadMeta(entry.Name())
		if err != nil {
			log.Printf("warning: skipping user template %q: %v", entry.Name(), err)
			continue
		}
		result = append(result, domain.TemplateInfo{
			Name:        meta.Name,
			Description: meta.Description,
			Source:      "user",
		})
	}
	return result, nil
}
