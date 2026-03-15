package service_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/service"
)

// spyRunner records calls to ContainerRunner methods.
type spyRunner struct {
	findCalls []findCall
	execCalls []execCall
	logsCalls []logsCall

	findResult string
	findErr    error
	execErr    error
	logsResult io.ReadCloser
	logsErr    error
}

type findCall struct {
	project string
	service string
}

type execCall struct {
	containerName string
	cmd           []string
	tty           bool
}

type logsCall struct {
	containerName string
	follow        bool
	tail          string
}

func (f *spyRunner) FindContainer(project, svc string) (string, error) {
	f.findCalls = append(f.findCalls, findCall{project, svc})
	return f.findResult, f.findErr
}

func (f *spyRunner) Exec(containerName string, cmd []string, tty bool) error {
	f.execCalls = append(f.execCalls, execCall{containerName, cmd, tty})
	return f.execErr
}

func (f *spyRunner) Logs(containerName string, follow bool, tail string) (io.ReadCloser, error) {
	f.logsCalls = append(f.logsCalls, logsCall{containerName, follow, tail})
	return f.logsResult, f.logsErr
}

func newTestContainer(t *testing.T) (*service.ContainerService, *spyRunner) {
	t.Helper()
	runner := &spyRunner{findResult: "abc123"}
	svc := service.NewContainerService(runner)
	return svc, runner
}

func TestShell_FindsContainerAndExecsWithTTY(t *testing.T) {
	svc, runner := newTestContainer(t)

	if err := svc.Shell("myapp", "devcontainer", []string{"zsh"}); err != nil {
		t.Fatalf("Shell() error: %v", err)
	}

	// Verify FindContainer was called correctly.
	if len(runner.findCalls) != 1 {
		t.Fatalf("expected 1 FindContainer call, got %d", len(runner.findCalls))
	}
	fc := runner.findCalls[0]
	if fc.project != "myapp" {
		t.Errorf("FindContainer project = %q, want %q", fc.project, "myapp")
	}
	if fc.service != "devcontainer" {
		t.Errorf("FindContainer service = %q, want %q", fc.service, "devcontainer")
	}

	// Verify Exec was called with TTY and correct command.
	if len(runner.execCalls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(runner.execCalls))
	}
	ec := runner.execCalls[0]
	if ec.containerName != "abc123" {
		t.Errorf("Exec containerName = %q, want %q", ec.containerName, "abc123")
	}
	if len(ec.cmd) != 1 || ec.cmd[0] != "zsh" {
		t.Errorf("Exec cmd = %v, want [zsh]", ec.cmd)
	}
	if !ec.tty {
		t.Error("Exec tty = false, want true")
	}
}

func TestShell_CustomCommand(t *testing.T) {
	svc, runner := newTestContainer(t)

	if err := svc.Shell("myapp", "devcontainer", []string{"bash", "-l"}); err != nil {
		t.Fatalf("Shell() error: %v", err)
	}

	if len(runner.execCalls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(runner.execCalls))
	}
	ec := runner.execCalls[0]
	if len(ec.cmd) != 2 || ec.cmd[0] != "bash" || ec.cmd[1] != "-l" {
		t.Errorf("Exec cmd = %v, want [bash -l]", ec.cmd)
	}
}

func TestExec_FindsContainerAndExecsWithoutTTY(t *testing.T) {
	svc, runner := newTestContainer(t)

	cmd := []string{"go", "test", "./..."}
	if err := svc.Exec("myapp", "devcontainer", cmd); err != nil {
		t.Fatalf("Exec() error: %v", err)
	}

	// Verify FindContainer was called.
	if len(runner.findCalls) != 1 {
		t.Fatalf("expected 1 FindContainer call, got %d", len(runner.findCalls))
	}

	// Verify Exec was called without TTY.
	if len(runner.execCalls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(runner.execCalls))
	}
	ec := runner.execCalls[0]
	if ec.containerName != "abc123" {
		t.Errorf("Exec containerName = %q, want %q", ec.containerName, "abc123")
	}
	if len(ec.cmd) != 3 || ec.cmd[0] != "go" || ec.cmd[1] != "test" || ec.cmd[2] != "./..." {
		t.Errorf("Exec cmd = %v, want [go test ./...]", ec.cmd)
	}
	if ec.tty {
		t.Error("Exec tty = true, want false")
	}
}

func TestLogs_FindsContainerAndReturnsReader(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.logsResult = io.NopCloser(strings.NewReader("log line 1\n"))

	rc, err := svc.Logs("myapp", "devcontainer", true, "100")
	if err != nil {
		t.Fatalf("Logs() error: %v", err)
	}
	defer rc.Close()

	// Verify FindContainer was called.
	if len(runner.findCalls) != 1 {
		t.Fatalf("expected 1 FindContainer call, got %d", len(runner.findCalls))
	}
	fc := runner.findCalls[0]
	if fc.project != "myapp" || fc.service != "devcontainer" {
		t.Errorf("FindContainer(%q, %q), want (myapp, devcontainer)", fc.project, fc.service)
	}

	// Verify Logs was called with correct params.
	if len(runner.logsCalls) != 1 {
		t.Fatalf("expected 1 Logs call, got %d", len(runner.logsCalls))
	}
	lc := runner.logsCalls[0]
	if lc.containerName != "abc123" {
		t.Errorf("Logs containerName = %q, want %q", lc.containerName, "abc123")
	}
	if !lc.follow {
		t.Error("Logs follow = false, want true")
	}
	if lc.tail != "100" {
		t.Errorf("Logs tail = %q, want %q", lc.tail, "100")
	}

	// Verify we can read from the returned reader.
	data, _ := io.ReadAll(rc)
	if string(data) != "log line 1\n" {
		t.Errorf("Logs output = %q, want %q", string(data), "log line 1\n")
	}
}

func TestShell_ContainerNotFound(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.findResult = ""
	runner.findErr = errors.New("container not found for project \"myapp\" service \"devcontainer\"")

	err := svc.Shell("myapp", "devcontainer", []string{"zsh"})
	if err == nil {
		t.Fatal("expected error when container not found")
	}
	if !strings.Contains(err.Error(), "container not found") {
		t.Errorf("error should contain cause, got: %v", err)
	}

	// Exec should not be called.
	if len(runner.execCalls) != 0 {
		t.Error("Exec should not be called when FindContainer fails")
	}
}

func TestExec_ContainerNotFound(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.findResult = ""
	runner.findErr = errors.New("container not found")

	err := svc.Exec("myapp", "devcontainer", []string{"go", "test"})
	if err == nil {
		t.Fatal("expected error when container not found")
	}

	if len(runner.execCalls) != 0 {
		t.Error("Exec should not be called when FindContainer fails")
	}
}

func TestLogs_ContainerNotFound(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.findResult = ""
	runner.findErr = errors.New("container not found")

	rc, err := svc.Logs("myapp", "devcontainer", false, "50")
	if err == nil {
		t.Fatal("expected error when container not found")
	}
	if rc != nil {
		t.Error("reader should be nil when FindContainer fails")
	}

	if len(runner.logsCalls) != 0 {
		t.Error("Logs should not be called when FindContainer fails")
	}
}

func TestShell_ExecFails(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.execErr = errors.New("exec failed")

	err := svc.Shell("myapp", "devcontainer", []string{"zsh"})
	if err == nil {
		t.Fatal("expected error when exec fails")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Errorf("error should contain cause, got: %v", err)
	}
}

func TestExec_ExecFails(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.execErr = errors.New("exec failed")

	err := svc.Exec("myapp", "devcontainer", []string{"go", "test"})
	if err == nil {
		t.Fatal("expected error when exec fails")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Errorf("error should contain cause, got: %v", err)
	}
}

func TestLogs_LogsFails(t *testing.T) {
	svc, runner := newTestContainer(t)
	runner.logsErr = errors.New("logs failed")

	rc, err := svc.Logs("myapp", "devcontainer", true, "all")
	if err == nil {
		t.Fatal("expected error when logs fails")
	}
	if rc != nil {
		t.Error("reader should be nil when Logs fails")
	}
	if !strings.Contains(err.Error(), "logs failed") {
		t.Errorf("error should contain cause, got: %v", err)
	}
}
