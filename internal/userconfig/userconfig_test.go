package userconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	// Override HOME so Save()/Load() use a temp directory instead of the
	// real ~/.vox/config.yaml.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := Config{
		AIPostProcess: BoolPtr(true),
		PromptMode:    BoolPtr(false),
		AIModel:       StringPtr("claude-sonnet-4-20250514"),
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify the file was created with restrictive permissions.
	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}

	// Verify directory permissions.
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("config dir permissions = %o, want 0700", perm)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.AIPostProcess == nil || *loaded.AIPostProcess != true {
		t.Error("AIPostProcess should be true")
	}
	if loaded.PromptMode == nil || *loaded.PromptMode != false {
		t.Error("PromptMode should be false")
	}
	if loaded.VoiceCommands != nil {
		t.Error("VoiceCommands should be nil (not set)")
	}
	if loaded.AIModel == nil || *loaded.AIModel != "claude-sonnet-4-20250514" {
		t.Errorf("AIModel = %v, want claude-sonnet-4-20250514", loaded.AIModel)
	}
}

func TestLoadMissing(t *testing.T) {
	// Point HOME at a temp dir with no config file.
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AIPostProcess != nil {
		t.Error("missing file should return all-nil config")
	}
}

func TestOmitEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := Config{AIPostProcess: BoolPtr(true)}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path, _ := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "prompt_mode") {
		t.Error("nil fields should be omitted from YAML")
	}
	if !strings.Contains(s, "ai_postprocess") {
		t.Error("set fields should appear in YAML")
	}
}

func TestBoolPtr(t *testing.T) {
	p := BoolPtr(true)
	if p == nil || *p != true {
		t.Error("BoolPtr(true) failed")
	}
}

func TestStringPtr(t *testing.T) {
	p := StringPtr("test")
	if p == nil || *p != "test" {
		t.Error("StringPtr(test) failed")
	}
}
