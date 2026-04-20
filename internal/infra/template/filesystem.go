package template

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/templates"
)

// FilesystemSource loads templates from a directory on disk (e.g.
// ~/.config/deckhand/templates/). It implements the same listing interface as
// EmbeddedSource but reads from the real filesystem, allowing user-provided
// and overridden templates.
type FilesystemSource struct {
	Dir         string // root directory containing template subdirectories
	SourceLabel string // label for TemplateInfo.Source (e.g. "user", "local"); defaults to "user"
}

// Load reads the raw Dockerfile and compose template strings for the given
// template name from the filesystem directory. Returns fs.ErrNotExist only
// when the template directory itself is absent. Missing files within an
// existing template directory produce a distinct error so that compositeSource
// does not silently fall through on incomplete overrides.
//
// The compose template is resolved with a fallback: if the template directory
// contains its own compose.yaml.tmpl that file is used; otherwise the shared
// compose.yaml.tmpl from the embedded filesystem is used.
func (f *FilesystemSource) Load(name string) (dockerfile string, compose string, err error) {
	base, err := f.templateDir(name)
	if err != nil {
		return "", "", err
	}

	df, err := os.ReadFile(filepath.Join(base, "Dockerfile.tmpl"))
	if err != nil {
		return "", "", f.fileError("Dockerfile template", name, err)
	}

	cf, err := os.ReadFile(filepath.Join(base, "compose.yaml.tmpl"))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", "", f.fileError("compose template", name, err)
		}
		// Fall back to the shared compose template from the embedded filesystem.
		shared, sErr := templates.FS.ReadFile("compose.yaml.tmpl")
		if sErr != nil {
			return "", "", f.fileError("compose template", name, sErr)
		}
		cf = shared
	}

	return string(df), string(cf), nil
}

// LoadMeta reads and parses metadata.yaml for the given template name from the
// filesystem directory. Same error semantics as Load.
func (f *FilesystemSource) LoadMeta(name string) (*domain.TemplateMeta, error) {
	base, err := f.templateDir(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(base, "metadata.yaml"))
	if err != nil {
		return nil, f.fileError("metadata", name, err)
	}

	var meta domain.TemplateMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing metadata for template %q: %w", name, err)
	}

	return &meta, nil
}

// fileError wraps a file read error. If the underlying error is fs.ErrNotExist
// (file missing inside an existing template dir), it strips the ErrNotExist
// so compositeSource doesn't treat it as "template not found".
func (f *FilesystemSource) fileError(what, name string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("loading %s for template %q: file not found", what, name)
	}
	return fmt.Errorf("loading %s for template %q: %w", what, name, err)
}

// templateDir validates a template name and returns the full directory path.
// It rejects names containing path separators or traversal components.
// Returns fs.ErrNotExist (wrapped) when the template directory does not exist.
func (f *FilesystemSource) templateDir(name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || clean != filepath.Base(clean) || strings.ContainsRune(clean, os.PathSeparator) {
		return "", fmt.Errorf("invalid template name %q", name)
	}
	dir := filepath.Join(f.Dir, clean)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}
	return dir, nil
}

// List returns TemplateInfo for every template in the directory that has a
// valid metadata.yaml. If the directory does not exist, it returns an empty
// slice (not an error). Templates with missing or bad metadata are skipped
// with a log warning.
func (f *FilesystemSource) sourceLabel() string {
	if f.SourceLabel != "" {
		return f.SourceLabel
	}
	return "user"
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

	label := f.sourceLabel()
	var result []domain.TemplateInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		meta, err := f.LoadMeta(entry.Name())
		if err != nil {
			log.Printf("warning: skipping %s template %q: %v", label, entry.Name(), err)
			continue
		}
		result = append(result, domain.TemplateInfo{
			Name:        meta.Name,
			Description: meta.Description,
			Source:      label,
		})
	}
	return result, nil
}
