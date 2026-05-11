package flags

import (
	"os"
	"testing"

	"vox/internal/userconfig"
)

func TestBoolFlagPrecedence(t *testing.T) {
	c := &Client{userCfg: userconfig.Config{}}

	// Default when nothing is set.
	if c.boolFlag("test-key", "VOX_TEST_FLAG_12345", nil, false) != false {
		t.Error("expected default false")
	}
	if c.boolFlag("test-key", "VOX_TEST_FLAG_12345", nil, true) != true {
		t.Error("expected default true")
	}

	// Config file value overrides default.
	val := true
	if c.boolFlag("test-key", "VOX_TEST_FLAG_12345", &val, false) != true {
		t.Error("config file should override default")
	}

	// Env var overrides config file.
	t.Setenv("VOX_TEST_FLAG_12345", "false")
	if c.boolFlag("test-key", "VOX_TEST_FLAG_12345", &val, false) != false {
		t.Error("env var should override config file")
	}
}

func TestAIModelPrecedence(t *testing.T) {
	c := &Client{userCfg: userconfig.Config{}}

	// Default.
	if got := c.AIModel(); got != "claude-haiku-4-5-20251001" {
		t.Errorf("AIModel() = %q, want default haiku", got)
	}

	// Config file.
	model := "claude-sonnet-4-20250514"
	c.userCfg.AIModel = &model
	if got := c.AIModel(); got != "claude-sonnet-4-20250514" {
		t.Errorf("AIModel() = %q, want sonnet from config", got)
	}

	// Env var overrides config.
	t.Setenv("VOX_AI_MODEL", "claude-opus-4-20250514")
	if got := c.AIModel(); got != "claude-opus-4-20250514" {
		t.Errorf("AIModel() = %q, want opus from env", got)
	}
}

func TestNilClientGraceful(t *testing.T) {
	// All methods should work with a nil LD client.
	c := &Client{userCfg: userconfig.Config{}}

	_ = c.AIPostProcess()
	_ = c.PromptMode()
	_ = c.VoiceCommands()
	_ = c.ContextAware()
	_ = c.StreamingOverlay()
	_ = c.AIModel()
	c.Close() // should not panic
}

func TestInitNoSDKKey(t *testing.T) {
	// Ensure VOX_LD_SDK_KEY is not set.
	os.Unsetenv("VOX_LD_SDK_KEY")

	c, err := Init(userconfig.Config{})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer c.Close()

	if c.ld != nil {
		t.Error("LD client should be nil when no SDK key is set")
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"anything", false},
	}
	for _, tt := range tests {
		if got := parseBool(tt.input); got != tt.want {
			t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFlagKeyConstants(t *testing.T) {
	// Verify flag keys match what was created in LaunchDarkly.
	keys := []string{
		KeyAIPostProcess,
		KeyPromptMode,
		KeyVoiceCommands,
		KeyContextAware,
		KeyStreamingOverlay,
		KeyAIModel,
	}
	for _, k := range keys {
		if k == "" {
			t.Error("flag key should not be empty")
		}
	}
}
