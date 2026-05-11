package prompt

import (
	"context"
	"fmt"
	"strings"
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
// and clipboard reader.
func NewExecutor(claude Claude, clipboard ClipboardReader) *Executor {
	return &Executor{claude: claude, clipboard: clipboard}
}

// Execute runs a prompt-mode action and returns the result text.
// The action and subject come from the intent classifier.
func (e *Executor) Execute(ctx context.Context, action, subject, rawArgs string) (string, error) {
	switch action {
	case "summarize":
		return e.summarize(ctx, subject, rawArgs)
	case "explain":
		return e.explain(ctx, subject, rawArgs)
	case "rewrite":
		return e.rewrite(ctx, subject, rawArgs)
	case "translate":
		return e.translate(ctx, rawArgs)
	case "fix-grammar":
		return e.fixGrammar(ctx)
	case "proofread":
		return e.proofread(ctx, subject)
	default:
		return "", fmt.Errorf("unknown prompt action: %q", action)
	}
}

func (e *Executor) summarize(ctx context.Context, subject, rawArgs string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}
	if content == "" && rawArgs != "" {
		content = rawArgs
	}

	return e.claude.Complete(ctx,
		"You are a concise summarizer. Summarize the following content in a few sentences. Return ONLY the summary.",
		content,
	)
}

func (e *Executor) explain(ctx context.Context, subject, rawArgs string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}
	if content == "" && rawArgs != "" {
		content = rawArgs
	}

	system := "You are a helpful technical explainer. Explain the following clearly and concisely. Return ONLY the explanation."
	if subject == "error" {
		system = "You are a debugging assistant. Explain the following error message: what it means, likely causes, and how to fix it. Be concise."
	}

	return e.claude.Complete(ctx, system, content)
}

func (e *Executor) rewrite(ctx context.Context, subject, rawArgs string) (string, error) {
	content, err := e.resolveContent(subject)
	if err != nil {
		return "", err
	}

	instruction := "Rewrite the following text."
	if rawArgs != "" {
		instruction = fmt.Sprintf("Rewrite the following text as %s.", rawArgs)
	}

	return e.claude.Complete(ctx,
		instruction+" Return ONLY the rewritten text with no explanation.",
		content,
	)
}

func (e *Executor) translate(ctx context.Context, rawArgs string) (string, error) {
	content, err := e.readClipboard()
	if err != nil {
		return "", err
	}

	lang := "the target language"
	if rawArgs != "" {
		lang = rawArgs
	}

	return e.claude.Complete(ctx,
		fmt.Sprintf("Translate the following text to %s. Return ONLY the translation.", lang),
		content,
	)
}

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
// is "clipboard", "this", or empty, it reads from the clipboard.
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
	return content, nil
}
