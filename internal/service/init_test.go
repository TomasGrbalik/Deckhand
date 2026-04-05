package service_test

import (
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

func newInitService(templates []domain.TemplateInfo, source *fakeSource) *service.InitService {
	lister := &fakeTemplateLister{templates: templates}
	return service.NewInitService(lister, source, nil)
}

func TestInitService_ListTemplates(t *testing.T) {
	templates := []domain.TemplateInfo{
		{Name: "base", Description: "Minimal", Source: "builtin"},
		{Name: "python", Description: "Python", Source: "builtin"},
	}
	svc := newInitService(templates, newFakeSource())

	result, err := svc.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates() error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(result))
	}
}

func TestInitService_ResolveTemplate_Found(t *testing.T) {
	templates := []domain.TemplateInfo{
		{Name: "base", Description: "Minimal", Source: "builtin"},
		{Name: "python", Description: "Python", Source: "builtin"},
	}
	source := &fakeSource{
		meta: &domain.TemplateMeta{
			Name:        "python",
			Description: "Python dev container",
			Variables: map[string]domain.TemplateVariable{
				"python_version": {Default: "3.12", Description: "Python version"},
			},
		},
	}
	svc := newInitService(templates, source)

	meta, err := svc.ResolveTemplate("python")
	if err != nil {
		t.Fatalf("ResolveTemplate() error: %v", err)
	}

	if meta.Name != "python" {
		t.Errorf("expected name %q, got %q", "python", meta.Name)
	}
	if _, ok := meta.Variables["python_version"]; !ok {
		t.Error("expected python_version variable in metadata")
	}
}

func TestInitService_ResolveTemplate_NotFound(t *testing.T) {
	templates := []domain.TemplateInfo{
		{Name: "base", Description: "Minimal", Source: "builtin"},
	}
	svc := newInitService(templates, newFakeSource())

	_, err := svc.ResolveTemplate("doesntexist")
	if err == nil {
		t.Fatal("expected error for unknown template, got nil")
	}

	// Should suggest `deckhand template list`.
	if got := err.Error(); !contains(got, "doesntexist") || !contains(got, "deckhand template list") {
		t.Errorf("error should mention template name and suggest template list, got: %v", err)
	}
}

func TestInitService_DefaultVariables(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{
			"go_version":  {Default: "1.23", Description: "Go version"},
			"air_version": {Default: "1.52", Description: "Air version"},
		},
	}

	defaults := svc.DefaultVariables(meta)

	if defaults["go_version"] != "1.23" {
		t.Errorf("go_version = %q, want %q", defaults["go_version"], "1.23")
	}
	if defaults["air_version"] != "1.52" {
		t.Errorf("air_version = %q, want %q", defaults["air_version"], "1.52")
	}
}

func TestInitService_DefaultVariables_Empty(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	defaults := svc.DefaultVariables(meta)
	if len(defaults) != 0 {
		t.Errorf("expected 0 defaults, got %d", len(defaults))
	}
}

func TestInitService_SortedVariableNames(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{
			"zebra": {Default: "z"},
			"alpha": {Default: "a"},
			"mike":  {Default: "m"},
		},
	}

	names := svc.SortedVariableNames(meta)

	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "mike" || names[2] != "zebra" {
		t.Errorf("expected sorted [alpha mike zebra], got %v", names)
	}
}

func TestInitService_BuildProject_WithOverrides(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{
			"python_version": {Default: "3.12"},
			"debug":          {Default: "false"},
		},
	}

	variables := map[string]string{
		"python_version": "3.11",  // overridden
		"debug":          "false", // same as default
	}

	proj := svc.BuildProject("my-api", "python", variables, meta, nil)

	if proj.Name != "my-api" {
		t.Errorf("Name = %q, want %q", proj.Name, "my-api")
	}
	if proj.Template != "python" {
		t.Errorf("Template = %q, want %q", proj.Template, "python")
	}

	// Only overrides (values differing from default) should be stored.
	if len(proj.Variables) != 1 {
		t.Fatalf("expected 1 variable override, got %d: %v", len(proj.Variables), proj.Variables)
	}
	if proj.Variables["python_version"] != "3.11" {
		t.Errorf("python_version = %q, want %q", proj.Variables["python_version"], "3.11")
	}
}

func TestInitService_BuildProject_AllDefaults(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{
			"python_version": {Default: "3.12"},
		},
	}

	variables := map[string]string{
		"python_version": "3.12", // same as default
	}

	proj := svc.BuildProject("my-api", "python", variables, meta, nil)

	// No overrides → Variables should be nil (omitempty in YAML).
	if proj.Variables != nil {
		t.Errorf("expected nil Variables when all match defaults, got %v", proj.Variables)
	}
}

func TestInitService_BuildProject_SetsVersion(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	proj := svc.BuildProject("my-app", "base", map[string]string{}, meta, nil)

	if proj.Version != 1 {
		t.Errorf("Version = %d, want 1", proj.Version)
	}
}

func TestInitService_BuildProject_NoVariables(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	proj := svc.BuildProject("my-app", "base", map[string]string{}, meta, nil)

	if proj.Name != "my-app" {
		t.Errorf("Name = %q, want %q", proj.Name, "my-app")
	}
	if proj.Template != "base" {
		t.Errorf("Template = %q, want %q", proj.Template, "base")
	}
	if proj.Variables != nil {
		t.Errorf("expected nil Variables for template with no variables, got %v", proj.Variables)
	}
}

// fakeCompanionLister implements service.CompanionLister for testing.
type fakeCompanionLister struct {
	services []domain.CompanionService
}

func (f *fakeCompanionLister) ListAvailable() []domain.CompanionService {
	return f.services
}

func newInitServiceWithCompanions(companions *fakeCompanionLister) *service.InitService {
	return service.NewInitService(&fakeTemplateLister{}, newFakeSource(), companions)
}

func TestInitService_ListCompanions(t *testing.T) {
	companions := &fakeCompanionLister{
		services: []domain.CompanionService{
			{Name: "postgres", Description: "PostgreSQL"},
			{Name: "redis", Description: "Redis"},
		},
	}
	svc := newInitServiceWithCompanions(companions)

	result := svc.ListCompanions()

	if len(result) != 2 {
		t.Fatalf("expected 2 companions, got %d", len(result))
	}
	if result[0].Name != "postgres" {
		t.Errorf("first companion = %q, want %q", result[0].Name, "postgres")
	}
	if result[1].Name != "redis" {
		t.Errorf("second companion = %q, want %q", result[1].Name, "redis")
	}
}

func TestInitService_ListCompanions_NilLister(t *testing.T) {
	svc := service.NewInitService(&fakeTemplateLister{}, newFakeSource(), nil)

	result := svc.ListCompanions()

	if result != nil {
		t.Errorf("expected nil when no companion lister, got %v", result)
	}
}

func TestInitService_BuildProject_WithServices(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	proj := svc.BuildProject("my-app", "base", map[string]string{}, meta, []string{"postgres", "redis"})

	if len(proj.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(proj.Services))
	}
	if proj.Services[0].Name != "postgres" || !proj.Services[0].Enabled {
		t.Errorf("services[0] = %+v, want postgres enabled", proj.Services[0])
	}
	if proj.Services[1].Name != "redis" || !proj.Services[1].Enabled {
		t.Errorf("services[1] = %+v, want redis enabled", proj.Services[1])
	}
}

func TestInitService_BuildProject_NoServices(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	proj := svc.BuildProject("my-app", "base", map[string]string{}, meta, nil)

	if proj.Services != nil {
		t.Errorf("expected nil Services when none selected, got %v", proj.Services)
	}
}

func TestInitService_BuildProject_EmptyServices(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	proj := svc.BuildProject("my-app", "base", map[string]string{}, meta, []string{})

	// Empty slice should behave like nil — no services key in YAML.
	if proj.Services != nil {
		t.Errorf("expected nil Services when empty slice passed, got %v", proj.Services)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
