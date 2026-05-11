package userconfig

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{
		AIPostProcess: BoolPtr(true),
		PromptMode:    BoolPtr(false),
		AIModel:       StringPtr("claude-sonnet-4-20250514"),
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var loaded Config
	if err := yaml.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
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
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if os.IsNotExist(err) {
		// Expected: missing file returns zero-value config.
		var cfg Config
		if cfg.AIPostProcess != nil {
			t.Error("missing file should return all-nil config")
		}
		return
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
}

func TestOmitEmpty(t *testing.T) {
	cfg := Config{AIPostProcess: BoolPtr(true)}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if contains(s, "prompt_mode") {
		t.Error("nil fields should be omitted from YAML")
	}
	if !contains(s, "ai_postprocess") {
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

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
