package config_test

import (
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
