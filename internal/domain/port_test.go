package domain

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPortMappingYAMLRoundTrip(t *testing.T) {
	original := PortMapping{
		Port:     8080,
		Name:     "code-server",
		Protocol: "http",
		Internal: false,
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded PortMapping
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Port != original.Port {
		t.Errorf("Port: got %d, want %d", decoded.Port, original.Port)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Protocol != original.Protocol {
		t.Errorf("Protocol: got %q, want %q", decoded.Protocol, original.Protocol)
	}
	if decoded.Internal != original.Internal {
		t.Errorf("Internal: got %v, want %v", decoded.Internal, original.Internal)
	}
}

func TestPortMappingInternalFlag(t *testing.T) {
	input := `
port: 5432
name: postgres
protocol: tcp
internal: true
`
	var pm PortMapping
	if err := yaml.Unmarshal([]byte(input), &pm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if pm.Port != 5432 {
		t.Errorf("Port: got %d, want 5432", pm.Port)
	}
	if pm.Internal != true {
		t.Errorf("Internal: got %v, want true", pm.Internal)
	}
}
