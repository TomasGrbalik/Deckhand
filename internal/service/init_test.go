package service_test

import (
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

func newInitService(templates []domain.TemplateInfo, source *fakeSource) *service.InitService {
	lister := &fakeTemplateLister{templates: templates}
	return service.NewInitService(lister, source)
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

	proj := svc.BuildProject("my-api", "python", variables, meta)

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

	proj := svc.BuildProject("my-api", "python", variables, meta)

	// No overrides → Variables should be nil (omitempty in YAML).
	if proj.Variables != nil {
		t.Errorf("expected nil Variables when all match defaults, got %v", proj.Variables)
	}
}

func TestInitService_BuildProject_NoVariables(t *testing.T) {
	svc := newInitService(nil, newFakeSource())
	meta := &domain.TemplateMeta{
		Variables: map[string]domain.TemplateVariable{},
	}

	proj := svc.BuildProject("my-app", "base", map[string]string{}, meta)

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
