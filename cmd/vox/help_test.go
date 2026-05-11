package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestHelpOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary test in short mode")
	}

	cmd := exec.Command("go", "run", ".", "help")
	cmd.Dir = findProjectRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("vox help failed: %v\n%s", err, out)
	}

	output := string(out)
	checks := []string{
		"Usage: vox",
		"Commands:",
		"setup",
		"help",
		"version",
		"VOX_HOTKEY",
		"ANTHROPIC_API_KEY",
		"Quick Start",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q", check)
		}
	}
}

func TestVersionOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary test in short mode")
	}

	cmd := exec.Command("go", "run", ".", "version")
	cmd.Dir = findProjectRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("vox version failed: %v\n%s", err, out)
	}

	output := strings.TrimSpace(string(out))
	if !strings.Contains(output, "vox") || !strings.Contains(output, "2.0") {
		t.Errorf("version output unexpected: %q", output)
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up to find go.mod.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			return dir + "/cmd/vox"
		}
		parent := dir[:strings.LastIndex(dir, "/")]
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
