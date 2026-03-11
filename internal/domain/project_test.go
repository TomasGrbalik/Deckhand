package domain

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProjectYAMLRoundTrip(t *testing.T) {
	original := Project{
		Name:     "my-api",
		Template: "go",
		Ports: []PortMapping{
			{Port: 8080, Name: "http", Protocol: "http", Internal: false},
			{Port: 5432, Name: "postgres", Protocol: "tcp", Internal: true},
		},
		Env: map[string]string{
			"DATABASE_URL": "postgresql://dev:secret@postgres:5432/appdb",
			"DEBUG":        "true",
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Project
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Template != original.Template {
		t.Errorf("Template: got %q, want %q", decoded.Template, original.Template)
	}
	if len(decoded.Ports) != len(original.Ports) {
		t.Fatalf("Ports length: got %d, want %d", len(decoded.Ports), len(original.Ports))
	}
	if decoded.Ports[0].Port != 8080 {
		t.Errorf("Ports[0].Port: got %d, want 8080", decoded.Ports[0].Port)
	}
	if decoded.Ports[1].Internal != true {
		t.Errorf("Ports[1].Internal: got %v, want true", decoded.Ports[1].Internal)
	}
	if decoded.Env["DATABASE_URL"] != original.Env["DATABASE_URL"] {
		t.Errorf("Env[DATABASE_URL]: got %q, want %q", decoded.Env["DATABASE_URL"], original.Env["DATABASE_URL"])
	}
}

func TestProjectDeserializeFromConfig(t *testing.T) {
	input := `
project: test-project
template: base
ports:
  - port: 3000
    name: api
    protocol: http
  - port: 8080
    name: code-server
    protocol: http
env:
  APP_ENV: development
`
	var p Project
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if p.Name != "test-project" {
		t.Errorf("Name: got %q, want %q", p.Name, "test-project")
	}
	if p.Template != "base" {
		t.Errorf("Template: got %q, want %q", p.Template, "base")
	}
	if len(p.Ports) != 2 {
		t.Fatalf("Ports length: got %d, want 2", len(p.Ports))
	}
	if p.Ports[0].Name != "api" {
		t.Errorf("Ports[0].Name: got %q, want %q", p.Ports[0].Name, "api")
	}
	if p.Env["APP_ENV"] != "development" {
		t.Errorf("Env[APP_ENV]: got %q, want %q", p.Env["APP_ENV"], "development")
	}
}
