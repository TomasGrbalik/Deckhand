package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/config"
)

func TestProjectConfigPath(t *testing.T) {
	got := config.ProjectConfigPath("/home/user/myproject")
	want := "/home/user/myproject/.deckhand.yaml"
	if got != want {
		t.Errorf("ProjectConfigPath() = %q, want %q", got, want)
	}
}

func TestGeneratedDir(t *testing.T) {
	got := config.GeneratedDir("/home/user/myproject")
	want := "/home/user/myproject/.deckhand"
	if got != want {
		t.Errorf("GeneratedDir() = %q, want %q", got, want)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	path, err := config.GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() returned error: %v", err)
	}
	if path == "" {
		t.Fatal("GlobalConfigPath() returned empty string")
	}
	// Should end with the expected suffix regardless of platform.
	wantSuffix := filepath.Join("deckhand", "config.yaml")
	if !strings.HasSuffix(path, wantSuffix) {
		t.Errorf("GlobalConfigPath() = %q, want suffix %q", path, wantSuffix)
	}
}
