package cli

import (
	"bytes"
	"testing"
)

func TestRootCommandRegistersAllSubcommands(t *testing.T) {
	root := newRootCmd()

	expected := []string{"init", "up", "down", "destroy", "shell", "exec", "logs", "status", "list", "port", "connect"}
	registered := make(map[string]bool)
	for _, sub := range root.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Errorf("command %q not registered on root", name)
		}
	}
}

func TestRootCommandHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root --help: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("expected help output, got empty string")
	}
}

func TestSubcommandHelp(t *testing.T) {
	commands := []string{"init", "up", "down", "destroy", "shell", "exec", "logs", "status", "list", "port", "connect"}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			root := newRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetArgs([]string{name, "--help"})

			if err := root.Execute(); err != nil {
				t.Fatalf("%s --help: %v", name, err)
			}

			if buf.Len() == 0 {
				t.Fatalf("expected help output for %s, got empty", name)
			}
		})
	}
}

func TestInitFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"init", "--help"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("init --help: %v", err)
	}

	output := buf.String()
	for _, flag := range []string{"--template", "--project"} {
		if !bytes.Contains([]byte(output), []byte(flag)) {
			t.Errorf("init help missing flag %s", flag)
		}
	}
}

func TestUpFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"up", "--help"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("up --help: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("--build")) {
		t.Error("up help missing --build flag")
	}
}

func TestDestroyFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"destroy", "--help"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("destroy --help: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("--yes")) {
		t.Error("destroy help missing --yes flag")
	}
}

func TestShellFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"shell", "--help"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("shell --help: %v", err)
	}

	output := buf.String()
	for _, flag := range []string{"--service", "--cmd"} {
		if !bytes.Contains([]byte(output), []byte(flag)) {
			t.Errorf("shell help missing flag %s", flag)
		}
	}
}

func TestLogsFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"logs", "--help"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("logs --help: %v", err)
	}

	output := buf.String()
	for _, flag := range []string{"--follow", "--tail"} {
		if !bytes.Contains([]byte(output), []byte(flag)) {
			t.Errorf("logs help missing flag %s", flag)
		}
	}
}

func TestGlobalVerboseFlag(t *testing.T) {
	root := newRootCmd()

	f := root.PersistentFlags().Lookup("verbose")
	if f == nil {
		t.Fatal("missing --verbose persistent flag")
	}
	if f.Shorthand != "v" {
		t.Errorf("--verbose shorthand = %q, want %q", f.Shorthand, "v")
	}
}

func TestPortSubcommands(t *testing.T) {
	root := newRootCmd()
	portCmd, _, err := root.Find([]string{"port"})
	if err != nil {
		t.Fatalf("port command not found: %v", err)
	}

	subcommands := make(map[string]bool)
	for _, sub := range portCmd.Commands() {
		subcommands[sub.Name()] = true
	}

	for _, name := range []string{"list", "add", "remove"} {
		if !subcommands[name] {
			t.Errorf("port subcommand %q not registered", name)
		}
	}
}

func TestPortAddFlags(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"port", "add", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("port add --help: %v", err)
	}

	output := buf.String()
	for _, flag := range []string{"--name", "--protocol"} {
		if !bytes.Contains([]byte(output), []byte(flag)) {
			t.Errorf("port add help missing flag %s", flag)
		}
	}
}

func TestConnectFlags(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"connect", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("connect --help: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("--host")) {
		t.Error("connect help missing --host flag")
	}
}

func TestVersionFlag(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("--version: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected version output, got empty")
	}
}
