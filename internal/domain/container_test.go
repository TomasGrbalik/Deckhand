package domain

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestContainerYAMLRoundTrip(t *testing.T) {
	original := Container{
		ID:      "abc123def456",
		Name:    "deckhand-my-api-devcontainer-1",
		Service: "devcontainer",
		Status:  "running",
		Health:  "healthy",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Container
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Service != original.Service {
		t.Errorf("Service: got %q, want %q", decoded.Service, original.Service)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Health != original.Health {
		t.Errorf("Health: got %q, want %q", decoded.Health, original.Health)
	}
}
