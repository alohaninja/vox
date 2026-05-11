package commands

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const shellTimeout = 30 * time.Second

// ShellCommand executes a shell command and returns its output.
type ShellCommand struct {
	name    string
	actions []string
	builder func(args string) (string, []string) // returns command and args
}

// NewShellCommand creates a command that matches the given actions and
// builds a shell command from the args.
func NewShellCommand(name string, actions []string, builder func(args string) (string, []string)) *ShellCommand {
	return &ShellCommand{name: name, actions: actions, builder: builder}
}

func (c *ShellCommand) Name() string { return c.name }

func (c *ShellCommand) Match(action string) bool {
	for _, a := range c.actions {
		if a == action {
			return true
		}
	}
	return false
}

func (c *ShellCommand) Execute(ctx context.Context, args string) (string, error) {
	cmdName, cmdArgs := c.builder(args)

	ctx, cancel := context.WithTimeout(ctx, shellTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s: %s", c.name, errMsg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// DefaultCommands returns the built-in voice commands.
func DefaultCommands() []Command {
	return []Command{
		NewShellCommand("create-pr", []string{"create-pr"}, func(args string) (string, []string) {
			cmdArgs := []string{"pr", "create", "--fill"}
			if args != "" {
				cmdArgs = append(cmdArgs, "--title", args)
			}
			return "gh", cmdArgs
		}),

		NewShellCommand("list-issues", []string{"list-issues"}, func(args string) (string, []string) {
			return "gh", []string{"issue", "list", "--assignee", "@me", "--limit", "10"}
		}),

		NewShellCommand("query-flag", []string{"query-flag"}, func(args string) (string, []string) {
			if args == "" {
				return "echo", []string{"usage: query flag <flag-key>"}
			}
			// Use the LD CLI if available, otherwise curl the API.
			return "ldcli", []string{"flags", "get", "--flag", args, "--project", "default", "--environment", "production"}
		}),

		NewShellCommand("create-ticket", []string{"create-ticket"}, func(args string) (string, []string) {
			summary := "New ticket"
			if args != "" {
				summary = args
			}
			return "echo", []string{fmt.Sprintf("[TODO] Create Jira ticket: %s", summary)}
		}),

		NewShellCommand("open-url", []string{"open-url"}, func(args string) (string, []string) {
			if args == "" {
				return "echo", []string{"usage: open <url>"}
			}
			return "open", []string{args}
		}),
	}
}
