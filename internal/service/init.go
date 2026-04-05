package service

import (
	"fmt"
	"sort"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// CompanionLister returns the available companion services for selection.
// The CompanionRegistry satisfies this interface.
type CompanionLister interface {
	ListAvailable() []domain.CompanionService
}

// InitService handles the business logic for initializing a new project.
// It discovers templates, validates selections, and builds project configs.
// No UI/CLI imports — form interaction stays in the CLI layer.
type InitService struct {
	lister     TemplateLister
	source     TemplateSource
	companions CompanionLister
}

// NewInitService creates an InitService with the given template lister and source.
// An optional CompanionLister can be provided for companion service selection.
func NewInitService(lister TemplateLister, source TemplateSource, companions CompanionLister) *InitService {
	return &InitService{lister: lister, source: source, companions: companions}
}

// ListTemplates returns all available templates.
func (s *InitService) ListTemplates() ([]domain.TemplateInfo, error) {
	return s.lister.List()
}

// ResolveTemplate validates that a template exists and returns its metadata.
// Returns an error with a suggestion to run `deckhand template list` if not found.
func (s *InitService) ResolveTemplate(name string) (*domain.TemplateMeta, error) {
	templates, err := s.lister.List()
	if err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}

	found := false
	for _, t := range templates {
		if t.Name == name {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("unknown template %q (run `deckhand template list` to see available templates)", name)
	}

	meta, err := s.source.LoadMeta(name)
	if err != nil {
		return nil, fmt.Errorf("loading template metadata: %w", err)
	}

	return meta, nil
}

// DefaultVariables returns the template's variable defaults as a map.
// The keys are sorted for stable ordering in forms.
func (s *InitService) DefaultVariables(meta *domain.TemplateMeta) map[string]string {
	defaults := make(map[string]string, len(meta.Variables))
	for k, v := range meta.Variables {
		defaults[k] = v.Default
	}
	return defaults
}

// SortedVariableNames returns the template variable names in sorted order
// for stable form rendering.
func (s *InitService) SortedVariableNames(meta *domain.TemplateMeta) []string {
	names := make([]string, 0, len(meta.Variables))
	for k := range meta.Variables {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ListCompanions returns the available companion services for selection.
// Returns nil if no CompanionLister was provided.
func (s *InitService) ListCompanions() []domain.CompanionService {
	if s.companions == nil {
		return nil
	}
	return s.companions.ListAvailable()
}

// BuildProject creates a domain.Project from the given inputs.
// Variables are only included if they differ from template defaults.
// Selected service names produce ServiceConfig entries with Enabled=true.
func (s *InitService) BuildProject(projectName, templateName string, variables map[string]string, meta *domain.TemplateMeta, selectedServices []string) *domain.Project {
	// Only store variables that differ from defaults.
	overrides := make(map[string]string)
	for k, v := range variables {
		if tmplVar, ok := meta.Variables[k]; ok && v != tmplVar.Default {
			overrides[k] = v
		}
	}

	proj := &domain.Project{
		Version:  1,
		Name:     projectName,
		Template: templateName,
	}
	if len(overrides) > 0 {
		proj.Variables = overrides
	}

	if len(selectedServices) > 0 {
		services := make([]domain.ServiceConfig, len(selectedServices))
		for i, name := range selectedServices {
			services[i] = domain.ServiceConfig{
				Name:    name,
				Enabled: true,
			}
		}
		proj.Services = services
	}

	return proj
}
