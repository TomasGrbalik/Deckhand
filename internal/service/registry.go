package service

import "github.com/TomasGrbalik/deckhand/internal/domain"

// TemplateLister can enumerate all templates it knows about.
type TemplateLister interface {
	List() ([]domain.TemplateInfo, error)
}

// TemplateRegistry merges templates from multiple sources. When the same
// template name appears in more than one source, the last source wins — so
// pass the user source after the builtin source to get user-overrides-builtin
// semantics.
type TemplateRegistry struct {
	sources []TemplateLister
}

// NewTemplateRegistry creates a registry that queries the given sources in
// order. Later sources override earlier ones when template names collide.
func NewTemplateRegistry(sources ...TemplateLister) *TemplateRegistry {
	return &TemplateRegistry{sources: sources}
}

// List returns a merged, deduplicated list of templates. When two sources
// provide a template with the same name, the one from the later source wins.
func (r *TemplateRegistry) List() ([]domain.TemplateInfo, error) {
	seen := make(map[string]int) // name → index in result
	var result []domain.TemplateInfo

	for _, src := range r.sources {
		templates, err := src.List()
		if err != nil {
			return nil, err
		}
		for _, t := range templates {
			if idx, exists := seen[t.Name]; exists {
				result[idx] = t // override with later source
			} else {
				seen[t.Name] = len(result)
				result = append(result, t)
			}
		}
	}

	return result, nil
}
