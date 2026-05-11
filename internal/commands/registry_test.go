package commands

import (
	"context"
	"errors"
	"testing"
)

type mockCommand struct {
	name     string
	action   string
	output   string
	err      error
	executed bool
	lastArgs string
}

func (m *mockCommand) Name() string             { return m.name }
func (m *mockCommand) Match(action string) bool { return action == m.action }
func (m *mockCommand) Execute(_ context.Context, args string) (string, error) {
	m.executed = true
	m.lastArgs = args
	return m.output, m.err
}

func TestRegistryExecute(t *testing.T) {
	cmd := &mockCommand{name: "test", action: "test-action", output: "done"}
	r := NewRegistry(cmd)

	result, err := r.Execute(context.Background(), "test-action", "some args")
	if err != nil {
		t.Fatal(err)
	}
	if result != "done" {
		t.Errorf("got %q, want %q", result, "done")
	}
	if !cmd.executed {
		t.Error("command should have been executed")
	}
	if cmd.lastArgs != "some args" {
		t.Errorf("args = %q, want %q", cmd.lastArgs, "some args")
	}
}

func TestRegistryNoMatch(t *testing.T) {
	cmd := &mockCommand{name: "test", action: "test-action"}
	r := NewRegistry(cmd)

	_, err := r.Execute(context.Background(), "unknown-action", "")
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestRegistryFirstMatch(t *testing.T) {
	cmd1 := &mockCommand{name: "first", action: "do-thing", output: "first"}
	cmd2 := &mockCommand{name: "second", action: "do-thing", output: "second"}
	r := NewRegistry(cmd1, cmd2)

	result, _ := r.Execute(context.Background(), "do-thing", "")
	if result != "first" {
		t.Errorf("should use first matching command, got %q", result)
	}
	if cmd2.executed {
		t.Error("second command should not have been executed")
	}
}

func TestRegistryCommandError(t *testing.T) {
	cmd := &mockCommand{name: "fail", action: "fail", err: errors.New("boom")}
	r := NewRegistry(cmd)

	_, err := r.Execute(context.Background(), "fail", "")
	if err == nil {
		t.Fatal("expected error from command")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry(
		&mockCommand{name: "alpha"},
		&mockCommand{name: "beta"},
		&mockCommand{name: "gamma"},
	)
	names := r.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" || names[2] != "gamma" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestRegistryEmpty(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "anything", "")
	if err == nil {
		t.Fatal("empty registry should error")
	}
}
