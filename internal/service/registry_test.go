package service_test

import (
	"errors"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

type fakeTemplateLister struct {
	templates []domain.TemplateInfo
	err       error
}

func (f *fakeTemplateLister) List() ([]domain.TemplateInfo, error) {
	return f.templates, f.err
}

func TestRegistry_MergesSources(t *testing.T) {
	builtin := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "base", Description: "Base template", Source: "builtin"},
		{Name: "go", Description: "Go template", Source: "builtin"},
	}}
	user := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "rust", Description: "Rust template", Source: "user"},
	}}

	reg := service.NewTemplateRegistry(builtin, user)
	result, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(result))
	}

	names := map[string]string{}
	for _, t := range result {
		names[t.Name] = t.Source
	}
	if names["base"] != "builtin" {
		t.Errorf("base should be builtin, got %q", names["base"])
	}
	if names["go"] != "builtin" {
		t.Errorf("go should be builtin, got %q", names["go"])
	}
	if names["rust"] != "user" {
		t.Errorf("rust should be user, got %q", names["rust"])
	}
}

func TestRegistry_UserOverridesBuiltin(t *testing.T) {
	builtin := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "go", Description: "Builtin Go", Source: "builtin"},
	}}
	user := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "go", Description: "Custom Go", Source: "user"},
	}}

	reg := service.NewTemplateRegistry(builtin, user)
	result, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 template (deduplicated), got %d", len(result))
	}

	if result[0].Source != "user" {
		t.Errorf("overridden template source = %q, want \"user\"", result[0].Source)
	}
	if result[0].Description != "Custom Go" {
		t.Errorf("overridden template description = %q, want \"Custom Go\"", result[0].Description)
	}
}

func TestRegistry_LocalOverridesUserAndBuiltin(t *testing.T) {
	builtin := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "base", Description: "Builtin Base", Source: "builtin"},
		{Name: "go", Description: "Builtin Go", Source: "builtin"},
	}}
	user := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "base", Description: "User Base", Source: "user"},
	}}
	local := &fakeTemplateLister{templates: []domain.TemplateInfo{
		{Name: "base", Description: "Local Base", Source: "local"},
		{Name: "custom", Description: "Local Custom", Source: "local"},
	}}

	// Order: builtin → user → local (last wins).
	reg := service.NewTemplateRegistry(builtin, user, local)
	result, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(result))
	}

	names := map[string]string{}
	descs := map[string]string{}
	for _, tmpl := range result {
		names[tmpl.Name] = tmpl.Source
		descs[tmpl.Name] = tmpl.Description
	}

	// "base" should be overridden by local (last source).
	if names["base"] != "local" {
		t.Errorf("base source = %q, want \"local\"", names["base"])
	}
	if descs["base"] != "Local Base" {
		t.Errorf("base description = %q, want \"Local Base\"", descs["base"])
	}
	// "go" should remain builtin (no override).
	if names["go"] != "builtin" {
		t.Errorf("go source = %q, want \"builtin\"", names["go"])
	}
	// "custom" should be local.
	if names["custom"] != "local" {
		t.Errorf("custom source = %q, want \"local\"", names["custom"])
	}
}

func TestRegistry_EmptySources(t *testing.T) {
	reg := service.NewTemplateRegistry(&fakeTemplateLister{}, &fakeTemplateLister{})
	result, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 templates, got %d", len(result))
	}
}

func TestRegistry_SourceError(t *testing.T) {
	broken := &fakeTemplateLister{err: errors.New("disk on fire")}
	reg := service.NewTemplateRegistry(broken)

	_, err := reg.List()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
