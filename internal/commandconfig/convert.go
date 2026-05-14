package commandconfig

import (
	"fmt"
	"sort"
	"strings"

	"vox/internal/classify"
	"vox/internal/commands"
)

// ToCommand converts a CommandDef into a commands.Command.
func ToCommand(def CommandDef) commands.Command {
	return commands.NewShellCommand(def.Name, []string{def.Name}, func(args string) (string, []string) {
		cmdArgs := make([]string, len(def.Args))
		copy(cmdArgs, def.Args)

		if def.ForwardArgs && args != "" {
			if def.RequireDotSlash {
				if !strings.HasPrefix(args, "./") || strings.Contains(args, "..") {
					usage := fmt.Sprintf("usage: args must start with ./ and cannot contain .. (got %q)", args)
					if def.Usage != "" {
						usage = def.Usage
					}
					return "echo", []string{usage}
				}
			}
			cmdArgs = append(cmdArgs, strings.Fields(args)...)
		} else if def.ForwardArgs && args == "" && def.Usage != "" {
			return "echo", []string{def.Usage}
		}

		return def.Command, cmdArgs
	})
}

// Merge combines built-in commands with user-defined commands.
// User commands with the same name replace the built-in. New names are appended.
// Returns the merged command list and merged classifier prefix entries.
func Merge(builtins []commands.Command, userDefs []CommandDef) ([]commands.Command, []classify.PrefixEntry) {
	if len(userDefs) == 0 {
		return builtins, classify.DefaultCommandPrefixes()
	}

	// Build set of overridden names
	overridden := make(map[string]bool, len(userDefs))
	for _, d := range userDefs {
		overridden[d.Name] = true
	}

	// Keep non-overridden builtins
	var merged []commands.Command
	for _, cmd := range builtins {
		if !overridden[cmd.Name()] {
			merged = append(merged, cmd)
		}
	}

	// Append user commands (both overrides and new)
	for _, d := range userDefs {
		merged = append(merged, ToCommand(d))
	}

	// Build merged prefix list
	defaults := classify.DefaultCommandPrefixes()
	var prefixes []classify.PrefixEntry
	for _, p := range defaults {
		if !overridden[p.Action] {
			prefixes = append(prefixes, p)
		}
	}

	// Add user-defined prefixes
	for _, d := range userDefs {
		for _, trigger := range d.Triggers {
			prefixes = append(prefixes, classify.PrefixEntry{
				Prefix: trigger,
				Action: d.Name,
			})
		}
	}

	// Sort by prefix length descending
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i].Prefix) > len(prefixes[j].Prefix)
	})

	return merged, prefixes
}
