package template

import (
	"fmt"

	"github.com/TomasGrbalik/deckhand/templates"
)

// Load reads the raw Dockerfile and compose template strings for the given
// template name from the embedded filesystem. It does not render them —
// rendering is the responsibility of the service layer (#5).
func Load(name string) (dockerfile string, compose string, err error) {
	base := name

	df, err := templates.FS.ReadFile(base + "/Dockerfile.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("loading Dockerfile template %q: %w", name, err)
	}

	cf, err := templates.FS.ReadFile(base + "/compose.yaml.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("loading compose template %q: %w", name, err)
	}

	return string(df), string(cf), nil
}
