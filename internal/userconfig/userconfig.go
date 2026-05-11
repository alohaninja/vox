package userconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configDir = ".vox"
const configFile = "config.yaml"

// Config holds user preferences for Vox features.
// These can be overridden by env vars or LD flags.
type Config struct {
	AIPostProcess    *bool   `yaml:"ai_postprocess,omitempty"`
	PromptMode       *bool   `yaml:"prompt_mode,omitempty"`
	VoiceCommands    *bool   `yaml:"voice_commands,omitempty"`
	ContextAware     *bool   `yaml:"context_aware,omitempty"`
	StreamingOverlay *bool   `yaml:"streaming_overlay,omitempty"`
	AIModel          *string `yaml:"ai_model,omitempty"`
}

// Path returns the full path to the config file (~/.vox/config.yaml).
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads the config file. Returns an empty Config (all nil) if the
// file does not exist -- this is the normal case for new users.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config file, creating ~/.vox/ if needed.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// BoolPtr is a helper to create a *bool for config fields.
func BoolPtr(b bool) *bool { return &b }

// StringPtr is a helper to create a *string for config fields.
func StringPtr(s string) *string { return &s }
