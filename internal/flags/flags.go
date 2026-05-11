package flags

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	ld "github.com/launchdarkly/go-server-sdk/v7"
	"github.com/launchdarkly/go-server-sdk/v7/ldcomponents"

	"vox/internal/userconfig"
)

// Flag key constants -- must match the keys created in LaunchDarkly.
const (
	KeyAIPostProcess    = "vox-ai-postprocess"
	KeyPromptMode       = "vox-prompt-mode"
	KeyVoiceCommands    = "vox-voice-commands"
	KeyContextAware     = "vox-context-aware"
	KeyStreamingOverlay = "vox-streaming-overlay"
	KeyAIModel          = "vox-ai-model"
)

// Client wraps the LD SDK client with Vox-specific flag helpers.
// When the underlying LD client is nil (no SDK key), all evaluations
// fall back to the user config file and environment variables.
type Client struct {
	ld      *ld.LDClient
	ctx     ldcontext.Context
	userCfg userconfig.Config
}

// Init creates a Client. If VOX_LD_SDK_KEY is not set, the LD client
// is nil and all flags return values from the config file or env vars.
// This is the expected path for open-source users.
func Init(userCfg userconfig.Config) (*Client, error) {
	user := os.Getenv("USER")
	if user == "" {
		user = "anonymous"
	}

	ldCtx := ldcontext.NewBuilder(user).
		Kind("user").
		SetString("tool", "vox").
		Build()

	c := &Client{
		ctx:     ldCtx,
		userCfg: userCfg,
	}

	sdkKey := os.Getenv("VOX_LD_SDK_KEY")
	if sdkKey == "" {
		return c, nil
	}

	config := ld.Config{
		// Polling is better for short-lived CLI tools -- avoids holding
		// a streaming connection open.
		DataSource: ldcomponents.PollingDataSource().PollInterval(5 * time.Minute),
		// Skip event sending for a developer tool.
		Events: ldcomponents.NoEvents(),
	}

	ldClient, err := ld.MakeCustomClient(sdkKey, config, 3*time.Second)
	if err != nil {
		// Non-fatal: tool works without flags.
		return c, fmt.Errorf("LD client init (using defaults): %w", err)
	}
	c.ld = ldClient
	return c, nil
}

// Close shuts down the LD client if it was initialized.
func (c *Client) Close() {
	if c != nil && c.ld != nil {
		_ = c.ld.Close()
	}
}

// AIPostProcess returns whether AI post-processing is enabled.
// Precedence: LD flag > VOX_AI_POSTPROCESS env > config file > false.
func (c *Client) AIPostProcess() bool {
	return c.boolFlag(KeyAIPostProcess, "VOX_AI_POSTPROCESS", c.userCfg.AIPostProcess, false)
}

// PromptMode returns whether prompt mode is enabled.
func (c *Client) PromptMode() bool {
	return c.boolFlag(KeyPromptMode, "VOX_PROMPT_MODE", c.userCfg.PromptMode, false)
}

// VoiceCommands returns whether voice commands are enabled.
func (c *Client) VoiceCommands() bool {
	return c.boolFlag(KeyVoiceCommands, "VOX_VOICE_COMMANDS", c.userCfg.VoiceCommands, false)
}

// ContextAware returns whether context-aware formatting is enabled.
func (c *Client) ContextAware() bool {
	return c.boolFlag(KeyContextAware, "VOX_CONTEXT_AWARE", c.userCfg.ContextAware, false)
}

// StreamingOverlay returns whether the streaming overlay is enabled.
func (c *Client) StreamingOverlay() bool {
	return c.boolFlag(KeyStreamingOverlay, "VOX_STREAMING_OVERLAY", c.userCfg.StreamingOverlay, false)
}

// AIModel returns the Claude model to use.
// Precedence: LD flag > VOX_AI_MODEL env > config file > default.
func (c *Client) AIModel() string {
	const defaultModel = "claude-haiku-4-5-20251001"

	if c.ld != nil {
		val, _ := c.ld.StringVariationCtx(context.Background(), KeyAIModel, c.ctx, "")
		if val != "" {
			return val
		}
	}
	if v := os.Getenv("VOX_AI_MODEL"); v != "" {
		return v
	}
	if c.userCfg.AIModel != nil && *c.userCfg.AIModel != "" {
		return *c.userCfg.AIModel
	}
	return defaultModel
}

// boolFlag evaluates a boolean flag with the precedence chain:
// LD flag > env var > config file pointer > hardcoded default.
func (c *Client) boolFlag(key, envVar string, cfgVal *bool, defaultVal bool) bool {
	if c.ld != nil {
		val, _ := c.ld.BoolVariationCtx(context.Background(), key, c.ctx, defaultVal)
		return val
	}
	if v := os.Getenv(envVar); v != "" {
		return parseBool(v)
	}
	if cfgVal != nil {
		return *cfgVal
	}
	return defaultVal
}

func parseBool(s string) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}
