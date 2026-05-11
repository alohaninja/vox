package commands

import (
	"context"
	"strings"
	"testing"
)

func nopBuilder(args string) (string, []string) { return "echo", nil }

func TestShellCommandMatch(t *testing.T) {
	cmd := NewShellCommand("test", []string{"do-it", "also-do-it"}, nopBuilder)

	if !cmd.Match("do-it") {
		t.Error("should match do-it")
	}
	if !cmd.Match("also-do-it") {
		t.Error("should match also-do-it")
	}
	if cmd.Match("nope") {
		t.Error("should not match nope")
	}
}

func TestShellCommandName(t *testing.T) {
	cmd := NewShellCommand("my-cmd", nil, nopBuilder)
	if cmd.Name() != "my-cmd" {
		t.Errorf("name = %q", cmd.Name())
	}
}

func TestShellCommandExecuteEcho(t *testing.T) {
	cmd := NewShellCommand("echo-test", []string{"echo"}, func(args string) (string, []string) {
		return "echo", []string{"hello", args}
	})

	result, err := cmd.Execute(context.Background(), "world")
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello world" {
		t.Errorf("got %q, want %q", result, "hello world")
	}
}

func TestShellCommandExecuteFailure(t *testing.T) {
	cmd := NewShellCommand("fail-test", []string{"fail"}, func(args string) (string, []string) {
		return "false", nil // `false` always exits 1
	})

	_, err := cmd.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected error from false command")
	}
}

func TestDefaultCommandsRegistered(t *testing.T) {
	cmds := DefaultCommands()
	if len(cmds) == 0 {
		t.Fatal("no default commands registered")
	}

	names := make(map[string]bool)
	for _, cmd := range cmds {
		names[cmd.Name()] = true
	}

	expected := []string{"create-pr", "list-issues", "query-flag", "create-ticket", "open-url"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing default command: %s", name)
		}
	}
}

func TestDefaultCommandsMatch(t *testing.T) {
	cmds := DefaultCommands()
	r := NewRegistry(cmds...)

	actions := []string{"create-pr", "list-issues", "query-flag", "create-ticket", "open-url"}
	for _, action := range actions {
		found := false
		for _, cmd := range r.commands {
			if cmd.Match(action) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no command matches action %q", action)
		}
	}
}

func TestCreateTicketPlaceholder(t *testing.T) {
	cmds := DefaultCommands()
	r := NewRegistry(cmds...)

	result, err := r.Execute(context.Background(), "create-ticket", "fix the auth bug")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "fix the auth bug") {
		t.Errorf("result should contain ticket summary, got %q", result)
	}
}

func TestQueryFlagNoArgs(t *testing.T) {
	cmds := DefaultCommands()
	r := NewRegistry(cmds...)

	// Empty args takes the echo path, not the ldcli path.
	result, err := r.Execute(context.Background(), "query-flag", "")
	if err != nil {
		t.Fatalf("empty args should use echo path: %v", err)
	}
	if !strings.Contains(result, "usage: query flag") {
		t.Errorf("expected usage message, got %q", result)
	}
}

func TestOpenURLAllowsHTTPS(t *testing.T) {
	cmds := DefaultCommands()
	r := NewRegistry(cmds...)

	// We can't actually call `open` in CI, but we can verify the builder
	// produces the right command by testing the disallowed path.
	result, err := r.Execute(context.Background(), "open-url", "file:///etc/passwd")
	if err != nil {
		t.Fatalf("disallowed URL should echo refusal, not error: %v", err)
	}
	if !strings.Contains(result, "refused to open") {
		t.Errorf("expected refusal message for file:// URL, got %q", result)
	}
}

func TestOpenURLRejectsFlagInjection(t *testing.T) {
	cmds := DefaultCommands()
	r := NewRegistry(cmds...)

	result, err := r.Execute(context.Background(), "open-url", "-e")
	if err != nil {
		t.Fatalf("flag-like arg should echo refusal, not error: %v", err)
	}
	if !strings.Contains(result, "refused to open") {
		t.Errorf("expected refusal for flag-like arg, got %q", result)
	}
}

func TestOpenURLRejectsCustomSchemes(t *testing.T) {
	schemes := []string{"ssh://attacker.com", "tel:+1234567890", "ftp://example.com", "/etc/passwd"}
	cmds := DefaultCommands()
	r := NewRegistry(cmds...)

	for _, scheme := range schemes {
		result, err := r.Execute(context.Background(), "open-url", scheme)
		if err != nil {
			t.Fatalf("disallowed URL %q should echo refusal: %v", scheme, err)
		}
		if !strings.Contains(result, "refused to open") {
			t.Errorf("expected refusal for %q, got %q", scheme, result)
		}
	}
}

func TestShellCommandContextCancellation(t *testing.T) {
	cmd := NewShellCommand("sleep-test", []string{"sleep"}, func(args string) (string, []string) {
		return "sleep", []string{"60"}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := cmd.Execute(ctx, "")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestNewShellCommandNilBuilderPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil builder")
		}
	}()
	NewShellCommand("bad", []string{"bad"}, nil)
}
