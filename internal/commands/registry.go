package commands

import (
	"context"
	"fmt"
)

// Command represents a voice command that can be executed.
type Command interface {
	// Name returns the command's identifier (e.g., "create-pr").
	Name() string

	// Match returns true if this command handles the given action.
	Match(action string) bool

	// Execute runs the command and returns output text to inject.
	Execute(ctx context.Context, args string) (string, error)
}

// Registry holds registered voice commands and dispatches by action.
type Registry struct {
	commands []Command
}

// NewRegistry creates a Registry with the given commands.
func NewRegistry(cmds ...Command) *Registry {
	return &Registry{commands: cmds}
}

// Execute finds the first matching command for the action and runs it.
// Returns the output text and any error. If no command matches, returns
// an error.
func (r *Registry) Execute(ctx context.Context, action, args string) (string, error) {
	for _, cmd := range r.commands {
		if cmd.Match(action) {
			return cmd.Execute(ctx, args)
		}
	}
	return "", fmt.Errorf("no command registered for action %q", action)
}

// List returns the names of all registered commands.
func (r *Registry) List() []string {
	names := make([]string, len(r.commands))
	for i, cmd := range r.commands {
		names[i] = cmd.Name()
	}
	return names
}
