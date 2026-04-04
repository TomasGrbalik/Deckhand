package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestServiceConfigYAMLRoundTrip(t *testing.T) {
	original := []ServiceConfig{
		{Name: "postgres", Enabled: true},
		{Name: "redis", Version: "7", Enabled: true},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded []ServiceConfig
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("length: got %d, want 2", len(decoded))
	}
	if decoded[0].Name != "postgres" {
		t.Errorf("decoded[0].Name: got %q, want %q", decoded[0].Name, "postgres")
	}
	if decoded[1].Version != "7" {
		t.Errorf("decoded[1].Version: got %q, want %q", decoded[1].Version, "7")
	}
}

func TestServiceConfigVersionOmitEmpty(t *testing.T) {
	cfg := ServiceConfig{Name: "postgres", Enabled: true}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "version") {
		t.Errorf("marshaled YAML should not contain 'version' when empty\nGot:\n%s", data)
	}
}

func TestProjectServicesYAMLRoundTrip(t *testing.T) {
	original := Project{
		Name:     "my-api",
		Template: "go",
		Services: []ServiceConfig{
			{Name: "postgres", Enabled: true},
			{Name: "redis", Enabled: true},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(data), "services:") {
		t.Errorf("marshaled YAML should contain 'services:'\nGot:\n%s", data)
	}

	var decoded Project
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Services) != 2 {
		t.Fatalf("Services length: got %d, want 2", len(decoded.Services))
	}
	if decoded.Services[0].Name != "postgres" {
		t.Errorf("Services[0].Name: got %q, want %q", decoded.Services[0].Name, "postgres")
	}
}

func TestProjectServicesOmitEmpty(t *testing.T) {
	original := Project{
		Name:     "my-api",
		Template: "base",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if strings.Contains(string(data), "services") {
		t.Errorf("marshaled YAML should not contain 'services' when empty\nGot:\n%s", data)
	}
}

func TestCompanionServiceYAMLRoundTrip(t *testing.T) {
	original := CompanionService{
		Name:        "postgres",
		Description: "PostgreSQL relational database",
		Image:       "postgres:16-alpine",
		Ports:       []int{5432},
		Environment: map[string]string{"POSTGRES_USER": "dev"},
		HealthCheck: HealthCheck{
			Test:     "pg_isready -U dev",
			Interval: "5s",
			Timeout:  "3s",
			Retries:  5,
		},
		Volumes: []string{"postgres-data:/var/lib/postgresql/data"},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CompanionService
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Image != original.Image {
		t.Errorf("Image: got %q, want %q", decoded.Image, original.Image)
	}
	if len(decoded.Ports) != 1 || decoded.Ports[0] != 5432 {
		t.Errorf("Ports: got %v, want [5432]", decoded.Ports)
	}
	if decoded.HealthCheck.Test != original.HealthCheck.Test {
		t.Errorf("HealthCheck.Test: got %q, want %q", decoded.HealthCheck.Test, original.HealthCheck.Test)
	}
	if decoded.HealthCheck.Retries != 5 {
		t.Errorf("HealthCheck.Retries: got %d, want 5", decoded.HealthCheck.Retries)
	}
	if len(decoded.Volumes) != 1 {
		t.Errorf("Volumes length: got %d, want 1", len(decoded.Volumes))
	}
}
