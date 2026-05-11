package classify

import "testing"

func TestClassifyPromptMode(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMode   Mode
		wantAction string
	}{
		{"summarize clipboard", "summarize my clipboard", ModePrompt, "summarize"},
		{"summarize clipboard alt", "Summarize clipboard", ModePrompt, "summarize"},
		{"summarize this", "summarize this", ModePrompt, "summarize"},
		{"summarize freeform", "summarize the meeting notes from today", ModePrompt, "summarize"},
		{"explain error", "explain this error", ModePrompt, "explain"},
		{"explain this", "Explain this", ModePrompt, "explain"},
		{"explain freeform", "explain what happened in the last deploy", ModePrompt, "explain"},
		{"rewrite as commit", "rewrite this as a commit message", ModePrompt, "rewrite"},
		{"rewrite as", "rewrite as a haiku", ModePrompt, "rewrite"},
		{"translate to", "translate to Spanish", ModePrompt, "translate"},
		{"translate this to", "translate this to French", ModePrompt, "translate"},
		{"fix grammar", "fix the grammar", ModePrompt, "fix-grammar"},
		{"proofread", "proofread this", ModePrompt, "proofread"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got.Mode != tt.wantMode {
				t.Errorf("Classify(%q).Mode = %v, want %v", tt.input, got.Mode, tt.wantMode)
			}
			if got.Action != tt.wantAction {
				t.Errorf("Classify(%q).Action = %q, want %q", tt.input, got.Action, tt.wantAction)
			}
		})
	}
}

func TestClassifyCommandMode(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAction string
		wantArgs   string
	}{
		{"hey vox", "hey vox do something", "vox", "do something"},
		{"vox prefix", "vox check the logs", "vox", "check the logs"},
		{"create pr", "create a pull request for the bugfix", "create-pr", "for the bugfix"},
		{"create pr short", "create pr", "create-pr", ""},
		{"open pr", "open a PR", "create-pr", ""},
		{"list issues", "list my issues", "list-issues", ""},
		{"show issues", "show issues", "list-issues", ""},
		{"create ticket", "create a ticket for the auth bug", "create-ticket", "for the auth bug"},
		{"create jira", "create jira", "create-ticket", ""},
		{"query flag", "query flag enable-new-checkout", "query-flag", "enable-new-checkout"},
		{"check flag", "check flag dark-mode", "query-flag", "dark-mode"},
		{"flag status", "flag status my-feature", "query-flag", "my-feature"},
		{"open url", "open https://example.com", "open-url", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got.Mode != ModeCommand {
				t.Errorf("Classify(%q).Mode = %v, want command", tt.input, got.Mode)
			}
			if got.Action != tt.wantAction {
				t.Errorf("Classify(%q).Action = %q, want %q", tt.input, got.Action, tt.wantAction)
			}
			if got.RawArgs != tt.wantArgs {
				t.Errorf("Classify(%q).RawArgs = %q, want %q", tt.input, got.RawArgs, tt.wantArgs)
			}
		})
	}
}

func TestClassifyDictation(t *testing.T) {
	// These should all be classified as dictation -- they contain words that
	// look like commands but aren't in the right position.
	inputs := []string{
		"I want to explain something to my team",
		"Let me summarize what happened yesterday in my standup",
		"The summary of the meeting was good",
		"Can you help me create a presentation",
		"We should translate the requirements into code",
		"I was explaining the architecture to the new engineer",
		"There's a race condition in the drain path where WaitOrCancel interleaves with channel close during shutdown",
		"Hello world",
		"This is just regular dictation text",
		"",
		"The create PR workflow is broken",
		"I need to query the database for old records",
		"Check if the deployment is done",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			got := Classify(input)
			if got.Mode != ModeDictation {
				t.Errorf("Classify(%q) = %v (action=%q), want dictation", input, got.Mode, got.Action)
			}
		})
	}
}

func TestClassifyCaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  Mode
	}{
		{"SUMMARIZE MY CLIPBOARD", ModePrompt},
		{"Summarize My Clipboard", ModePrompt},
		{"CREATE PR", ModeCommand},
		{"Hey Vox do something", ModeCommand},
	}
	for _, tt := range tests {
		got := Classify(tt.input)
		if got.Mode != tt.want {
			t.Errorf("Classify(%q).Mode = %v, want %v", tt.input, got.Mode, tt.want)
		}
	}
}

func TestClassifyPreservesOriginalCase(t *testing.T) {
	got := Classify("Summarize My Important Meeting Notes")
	if got.RawArgs != "My Important Meeting Notes" {
		t.Errorf("RawArgs = %q, want original case preserved", got.RawArgs)
	}
}

func TestClassifyWhitespace(t *testing.T) {
	got := Classify("  summarize my clipboard  ")
	if got.Mode != ModePrompt {
		t.Errorf("leading/trailing whitespace should be trimmed; got mode %v", got.Mode)
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeDictation, "dictation"},
		{ModeCommand, "command"},
		{ModePrompt, "prompt"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("Mode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
