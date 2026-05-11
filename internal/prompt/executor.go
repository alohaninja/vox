package prompt

import (
	"context"
	"fmt"
	"strings"
)

// maxContentBytes is the maximum clipboard/input size (100 KB ≈ 25K tokens)
// that will be sent to the Claude API. Inputs exceeding this are rejected
// early to avoid wasted API calls and memory pressure.
const maxContentBytes = 100 * 1024

// Action represents a prompt-mode action dispatched by the intent classifier.
type Action string

const (
	ActionSummarize  Action = "summarize"
	ActionExplain    Action = "explain"
	ActionRewrite    Action = "rewrite"
	ActionTranslate  Action = "translate"
	ActionFixGrammar Action = "fix-grammar"
	ActionProofread  Action = "proofread"
)

// Claude is the interface for sending messages to Claude API.
// This decouples the prompt package from the concrete claude.Client.
type Claude interface {
	Complete(ctx context.Context, system, userMessage string) (string, error)
}

// ClipboardReader reads the current system clipboard contents.
type ClipboardReader func() (string, error)

// Executor handles prompt-mode actions by constructing Claude prompts
// and returning the AI-generated result.
type Executor struct {
	claude    Claude
	clipboard ClipboardReader
}

// NewExecutor creates a prompt executor with the given Claude client
// and clipboard reader. Both arguments must be non-nil.
func NewExecutor(claude Claude, clipboard ClipboardReader) *Executor {
	if claude == nil {
		panic("prompt.NewExecutor: claude must not be nil")
	}
	if clipboard == nil {
		panic("prompt.NewExecutor: clipboard must not be nil")
	}
	return &Executor{claude: claude, clipboard: clipboard}
}

// Execute runs a prompt-mode action and returns the result text.
// The action and subject come from the intent classifier.
func (e *Executor) Execute(ctx context.Context, action, subject, rawArgs string) (string, error) {
	switch Action(action) {
	case ActionSummarize:
		return e.summarize(ctx, subject)
	case ActionExplain:
		return e.explain(ctx, subject)
	case ActionRewrite:
		return e.rewrite(ctx, subject, rawArgs)
	case ActionTranslate:
		return e.translate(ctx, rawArgs)
	case ActionFixGrammar:
		return e.fixGrammar(ctx)
	case ActionProofread:
		return e.proofread(ctx, subject)
	default:
		return "", fmt.Errorf("unknown prompt action: %q", action)
	}
}

func (e *Executor) summarize(ctx context.Context, subject string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}

	return e.claude.Complete(ctx,
		"You are a concise summarizer. Summarize the following content in a few sentences. Return ONLY the summary.",
		content,
	)
}

func (e *Executor) explain(ctx context.Context, subject string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}

	system := "You are a helpful technical explainer. Explain the following clearly and concisely. Return ONLY the explanation."
	if strings.EqualFold(subject, "error") {
		system = "You are a debugging assistant. Explain the following error message: what it means, likely causes, and how to fix it. Be concise."
	}

	return e.claude.Complete(ctx, system, content)
}

func (e *Executor) rewrite(ctx context.Context, subject, rawArgs string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}

	// rawArgs (e.g. "a commit message") goes in the user message, not the
	// system prompt, to prevent prompt injection via voice transcription.
	system := "Rewrite the following text. Return ONLY the rewritten text with no explanation."

	if rawArgs != "" {
		return e.claude.Complete(ctx, system,
			fmt.Sprintf("Rewrite this as %s:\n\n%s", rawArgs, content),
		)
	}
	return e.claude.Complete(ctx, system, content)
}

// translate always reads from the clipboard because the subject (if present)
// is the target language, not the content to translate. This intentionally
// bypasses resolveContent.
func (e *Executor) translate(ctx context.Context, rawArgs string) (string, error) {
	content, err := e.readClipboard()
	if err != nil {
		return "", err
	}

	// Target language goes in the user message, not the system prompt,
	// to prevent prompt injection via voice transcription.
	system := "Translate the following text to the requested language. Return ONLY the translation."

	lang := "the target language"
	if rawArgs != "" {
		lang = rawArgs
	}

	return e.claude.Complete(ctx, system,
		fmt.Sprintf("Translate to %s:\n\n%s", lang, content),
	)
}

// fixGrammar always reads from the clipboard because there is no meaningful
// subject — the user simply says "fix grammar". This intentionally bypasses
// resolveContent.
func (e *Executor) fixGrammar(ctx context.Context) (string, error) {
	content, err := e.readClipboard()
	if err != nil {
		return "", err
	}

	return e.claude.Complete(ctx,
		"Fix the grammar, spelling, and punctuation in the following text. Preserve the original meaning and tone. Return ONLY the corrected text.",
		content,
	)
}

func (e *Executor) proofread(ctx context.Context, subject string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}

	return e.claude.Complete(ctx,
		"Proofread the following text. Fix any errors in grammar, spelling, punctuation, and style. Return ONLY the corrected text.",
		content,
	)
}

// resolveContent returns the content to operate on. If the subject
// is "clipboard", "this", "error", or empty, it reads from the clipboard.
// Otherwise, it returns the subject as literal text.
func (e *Executor) resolveContent(subject string) (string, error) {
	lower := strings.ToLower(subject)
	switch lower {
	case "clipboard", "this", "error", "":
		return e.readClipboard()
	default:
		return subject, nil
	}
}

func (e *Executor) readClipboard() (string, error) {
	content, err := e.clipboard()
	if err != nil {
		return "", fmt.Errorf("read clipboard: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("clipboard is empty")
	}
	if len(content) > maxContentBytes {
		return "", fmt.Errorf("clipboard content too large (%d bytes, max %d)", len(content), maxContentBytes)
	}
	return content, nil
}
