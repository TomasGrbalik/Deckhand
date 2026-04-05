package service_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

// fakeDockerChecker implements service.DockerChecker for testing.
type fakeDockerChecker struct {
	pingErr        error
	composeVersion string
	composeErr     error
}

func (f *fakeDockerChecker) Ping() error                     { return f.pingErr }
func (f *fakeDockerChecker) ComposeVersion() (string, error) { return f.composeVersion, f.composeErr }

// fakeConfigLoader implements service.ConfigLoader for testing.
type fakeConfigLoader struct {
	globalCfg  *domain.GlobalConfig
	globalErr  error
	projectCfg *domain.Project
	projectErr error
}

func (f *fakeConfigLoader) LoadGlobal() (*domain.GlobalConfig, error) {
	return f.globalCfg, f.globalErr
}

func (f *fakeConfigLoader) LoadProject(_ string) (*domain.Project, error) {
	return f.projectCfg, f.projectErr
}

// fakeTemplateSource implements service.TemplateSource for testing.
type fakeTemplateSource struct {
	available map[string]bool
}

func (f *fakeTemplateSource) Load(name string) (string, string, error) {
	if f.available[name] {
		return "FROM ubuntu", "services:", nil
	}
	return "", "", errors.New("template not found")
}

func (f *fakeTemplateSource) LoadMeta(name string) (*domain.TemplateMeta, error) {
	if f.available[name] {
		return &domain.TemplateMeta{}, nil
	}
	return nil, errors.New("template not found")
}

func TestDoctorAllPass(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{composeVersion: "2.24.0"},
		&fakeConfigLoader{
			globalCfg:  &domain.GlobalConfig{},
			projectCfg: &domain.Project{Name: "myapp", Template: "go"},
		},
		&fakeTemplateSource{available: map[string]bool{"go": true}},
	)

	results := svc.RunChecks("/tmp/myapp")

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != service.CheckPass {
			t.Errorf("%s: expected PASS, got %s (%s)", r.Name, r.Status, r.Message)
		}
	}
	if service.HasFailures(results) {
		t.Error("HasFailures() should be false when all checks pass")
	}
}

func TestDoctorDockerFail(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{pingErr: errors.New("cannot connect to daemon")},
		&fakeConfigLoader{
			globalCfg:  &domain.GlobalConfig{},
			projectCfg: &domain.Project{Name: "myapp", Template: "go"},
		},
		&fakeTemplateSource{available: map[string]bool{"go": true}},
	)

	results := svc.RunChecks("/tmp/myapp")

	if results[0].Status != service.CheckFail {
		t.Errorf("Docker daemon: expected FAIL, got %s", results[0].Status)
	}
	// All checks should still run.
	if len(results) != 5 {
		t.Fatalf("expected 5 results (no short-circuit), got %d", len(results))
	}
	if !service.HasFailures(results) {
		t.Error("HasFailures() should be true")
	}
}

func TestDoctorComposeFail(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{composeErr: errors.New("compose not found")},
		&fakeConfigLoader{
			globalCfg:  &domain.GlobalConfig{},
			projectCfg: &domain.Project{Name: "myapp", Template: "go"},
		},
		&fakeTemplateSource{available: map[string]bool{"go": true}},
	)

	results := svc.RunChecks("/tmp/myapp")

	if results[1].Status != service.CheckFail {
		t.Errorf("Compose V2: expected FAIL, got %s", results[1].Status)
	}
	if !service.HasFailures(results) {
		t.Error("HasFailures() should be true")
	}
}

func TestDoctorBadGlobalConfig(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{composeVersion: "2.24.0"},
		&fakeConfigLoader{
			globalErr:  errors.New("malformed YAML"),
			projectCfg: &domain.Project{Name: "myapp", Template: "go"},
		},
		&fakeTemplateSource{available: map[string]bool{"go": true}},
	)

	results := svc.RunChecks("/tmp/myapp")

	if results[2].Status != service.CheckFail {
		t.Errorf("Global config: expected FAIL, got %s", results[2].Status)
	}
}

func TestDoctorMissingProjectConfig(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{composeVersion: "2.24.0"},
		&fakeConfigLoader{
			globalCfg:  &domain.GlobalConfig{},
			projectErr: fs.ErrNotExist,
		},
		&fakeTemplateSource{available: map[string]bool{}},
	)

	results := svc.RunChecks("/tmp/noproject")

	// Project config should be SKIP, not FAIL.
	if results[3].Status != service.CheckSkip {
		t.Errorf("Project config: expected SKIP, got %s (%s)", results[3].Status, results[3].Message)
	}
	// Template should also be SKIP since no project was loaded.
	if results[4].Status != service.CheckSkip {
		t.Errorf("Template: expected SKIP, got %s (%s)", results[4].Status, results[4].Message)
	}
	// Neither SKIP should cause HasFailures to be true.
	if service.HasFailures(results) {
		t.Error("HasFailures() should be false when checks are only SKIP")
	}
}

func TestDoctorInvalidProjectConfig(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{composeVersion: "2.24.0"},
		&fakeConfigLoader{
			globalCfg:  &domain.GlobalConfig{},
			projectErr: errors.New("parsing config: invalid YAML"),
		},
		&fakeTemplateSource{available: map[string]bool{}},
	)

	results := svc.RunChecks("/tmp/badconfig")

	// Invalid config should be FAIL, not SKIP.
	if results[3].Status != service.CheckFail {
		t.Errorf("Project config: expected FAIL, got %s (%s)", results[3].Status, results[3].Message)
	}
	if !service.HasFailures(results) {
		t.Error("HasFailures() should be true for invalid project config")
	}
}

func TestDoctorTemplateMissing(t *testing.T) {
	svc := service.NewDoctorService(
		&fakeDockerChecker{composeVersion: "2.24.0"},
		&fakeConfigLoader{
			globalCfg:  &domain.GlobalConfig{},
			projectCfg: &domain.Project{Name: "myapp", Template: "nonexistent"},
		},
		&fakeTemplateSource{available: map[string]bool{}},
	)

	results := svc.RunChecks("/tmp/myapp")

	if results[4].Status != service.CheckFail {
		t.Errorf("Template: expected FAIL, got %s", results[4].Status)
	}
}
