package prompt

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockClaude records calls and returns a canned response.
type mockClaude struct {
	lastSystem string
	lastUser   string
	response   string
	err        error
}

func (m *mockClaude) Complete(_ context.Context, system, user string) (string, error) {
	m.lastSystem = system
	m.lastUser = user
	return m.response, m.err
}

func mockClipboard(content string, err error) ClipboardReader {
	return func() (string, error) { return content, err }
}

func TestSummarizeClipboard(t *testing.T) {
	mc := &mockClaude{response: "A short summary."}
	e := NewExecutor(mc, mockClipboard("long text here", nil))

	result, err := e.Execute(context.Background(), "summarize", "clipboard", "")
	if err != nil {
		t.Fatal(err)
	}
	if result != "A short summary." {
		t.Errorf("got %q", result)
	}
	if mc.lastUser != "long text here" {
		t.Errorf("expected clipboard content, got %q", mc.lastUser)
	}
}

func TestSummarizeFreeform(t *testing.T) {
	// The clipboard mock returns valid content here, but it should NOT be
	// called — the literal subject "the meeting notes" is not a keyword, so
	// resolveContent returns it directly.
	mc := &mockClaude{response: "Summary of notes."}
	e := NewExecutor(mc, mockClipboard("clipboard content", nil))

	result, err := e.Execute(context.Background(), "summarize", "the meeting notes", "the meeting notes")
	if err != nil {
		t.Fatal(err)
	}
	if result != "Summary of notes." {
		t.Errorf("got %q", result)
	}
	if mc.lastUser != "the meeting notes" {
		t.Errorf("expected literal text, got %q", mc.lastUser)
	}
}

func TestExplainError(t *testing.T) {
	mc := &mockClaude{response: "This error means..."}
	e := NewExecutor(mc, mockClipboard("panic: runtime error", nil))

	result, err := e.Execute(context.Background(), "explain", "error", "")
	if err != nil {
		t.Fatal(err)
	}
	if result != "This error means..." {
		t.Errorf("got %q", result)
	}
	if !strings.Contains(mc.lastSystem, "debugging") {
		t.Error("explain error should use debugging system prompt")
	}
}

func TestExplainErrorCaseInsensitive(t *testing.T) {
	mc := &mockClaude{response: "This error means..."}
	e := NewExecutor(mc, mockClipboard("panic: runtime error", nil))

	// "Error" (capitalized) should still trigger the debugging prompt.
	_, err := e.Execute(context.Background(), "explain", "Error", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mc.lastSystem, "debugging") {
		t.Error("explain with capitalized 'Error' should use debugging system prompt")
	}
}

func TestExplainGeneric(t *testing.T) {
	mc := &mockClaude{response: "Explanation."}
	e := NewExecutor(mc, mockClipboard("some code", nil))

	_, err := e.Execute(context.Background(), "explain", "this", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(mc.lastSystem, "debugging") {
		t.Error("generic explain should not use debugging prompt")
	}
}

func TestRewriteAsCommitMessage(t *testing.T) {
	mc := &mockClaude{response: "fix: handle nil pointer in auth"}
	e := NewExecutor(mc, mockClipboard("fixed the nil pointer bug in auth", nil))

	result, err := e.Execute(context.Background(), "rewrite", "this", "a commit message")
	if err != nil {
		t.Fatal(err)
	}
	if result != "fix: handle nil pointer in auth" {
		t.Errorf("got %q", result)
	}
	// rawArgs should appear in the user message, NOT the system prompt.
	if strings.Contains(mc.lastSystem, "commit message") {
		t.Error("rawArgs should not be in the system prompt (prompt injection risk)")
	}
	if !strings.Contains(mc.lastUser, "commit message") {
		t.Error("rawArgs should appear in the user message")
	}
	if !strings.Contains(mc.lastUser, "fixed the nil pointer bug in auth") {
		t.Error("clipboard content should appear in the user message")
	}
}

func TestRewriteNoArgs(t *testing.T) {
	mc := &mockClaude{response: "rewritten text"}
	e := NewExecutor(mc, mockClipboard("original text", nil))

	result, err := e.Execute(context.Background(), "rewrite", "clipboard", "")
	if err != nil {
		t.Fatal(err)
	}
	if result != "rewritten text" {
		t.Errorf("got %q", result)
	}
	if mc.lastUser != "original text" {
		t.Errorf("expected clipboard content as user message, got %q", mc.lastUser)
	}
}

func TestTranslate(t *testing.T) {
	mc := &mockClaude{response: "Hola mundo"}
	e := NewExecutor(mc, mockClipboard("Hello world", nil))

	result, err := e.Execute(context.Background(), "translate", "", "Spanish")
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hola mundo" {
		t.Errorf("got %q", result)
	}
	// Language should appear in user message, NOT system prompt.
	if strings.Contains(mc.lastSystem, "Spanish") {
		t.Error("target language should not be in the system prompt (prompt injection risk)")
	}
	if !strings.Contains(mc.lastUser, "Spanish") {
		t.Error("target language should appear in the user message")
	}
}

func TestTranslateNoLanguage(t *testing.T) {
	mc := &mockClaude{response: "translated"}
	e := NewExecutor(mc, mockClipboard("some text", nil))

	_, err := e.Execute(context.Background(), "translate", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mc.lastUser, "the target language") {
		t.Errorf("expected fallback language placeholder in user message, got %q", mc.lastUser)
	}
}

func TestFixGrammar(t *testing.T) {
	mc := &mockClaude{response: "This is correct."}
	e := NewExecutor(mc, mockClipboard("this is correctt", nil))

	result, err := e.Execute(context.Background(), "fix-grammar", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if result != "This is correct." {
		t.Errorf("got %q", result)
	}
}

func TestProofread(t *testing.T) {
	mc := &mockClaude{response: "Proofread text."}
	e := NewExecutor(mc, mockClipboard("proofred text", nil))

	_, err := e.Execute(context.Background(), "proofread", "this", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mc.lastSystem, "Proofread") {
		t.Error("proofread should use proofread system prompt")
	}
}

func TestUnknownAction(t *testing.T) {
	mc := &mockClaude{}
	e := NewExecutor(mc, mockClipboard("content", nil))

	_, err := e.Execute(context.Background(), "dance", "", "")
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestClipboardError(t *testing.T) {
	mc := &mockClaude{response: "ok"}
	e := NewExecutor(mc, mockClipboard("", errors.New("no clipboard")))

	_, err := e.Execute(context.Background(), "summarize", "clipboard", "")
	if err == nil {
		t.Fatal("expected error when clipboard fails")
	}
}

func TestEmptyClipboard(t *testing.T) {
	mc := &mockClaude{response: "ok"}
	e := NewExecutor(mc, mockClipboard("   ", nil))

	_, err := e.Execute(context.Background(), "summarize", "clipboard", "")
	if err == nil {
		t.Fatal("expected error for empty clipboard")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty clipboard, got: %v", err)
	}
}

func TestClipboardTooLarge(t *testing.T) {
	huge := strings.Repeat("x", maxContentBytes+1)
	mc := &mockClaude{response: "ok"}
	e := NewExecutor(mc, mockClipboard(huge, nil))

	_, err := e.Execute(context.Background(), "summarize", "clipboard", "")
	if err == nil {
		t.Fatal("expected error for oversized clipboard")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error should mention size, got: %v", err)
	}
}

func TestClaudeAPIError(t *testing.T) {
	mc := &mockClaude{err: errors.New("API rate limited")}
	e := NewExecutor(mc, mockClipboard("content", nil))

	_, err := e.Execute(context.Background(), "summarize", "clipboard", "")
	if err == nil {
		t.Fatal("expected error when Claude fails")
	}
}

func TestResolveContentLiteralText(t *testing.T) {
	mc := &mockClaude{response: "Summary of meeting notes."}
	e := NewExecutor(mc, mockClipboard("clipboard content", nil))

	_, err := e.Execute(context.Background(), "summarize", "the meeting notes from today", "the meeting notes from today")
	if err != nil {
		t.Fatal(err)
	}
	// Should use the literal subject, not clipboard
	if mc.lastUser != "the meeting notes from today" {
		t.Errorf("expected literal text, got %q", mc.lastUser)
	}
}

func TestNewExecutorNilClaude(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil claude")
		}
	}()
	NewExecutor(nil, mockClipboard("", nil))
}

func TestNewExecutorNilClipboard(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil clipboard")
		}
	}()
	NewExecutor(&mockClaude{}, nil)
}

func TestActionConstants(t *testing.T) {
	// Verify that the exported Action constants match the strings the
	// executor switch dispatches on.
	actions := []Action{
		ActionSummarize, ActionExplain, ActionRewrite,
		ActionTranslate, ActionFixGrammar, ActionProofread,
	}
	mc := &mockClaude{response: "ok"}
	e := NewExecutor(mc, mockClipboard("content", nil))

	for _, a := range actions {
		_, err := e.Execute(context.Background(), string(a), "clipboard", "")
		if err != nil {
			t.Errorf("action %q returned unexpected error: %v", a, err)
		}
	}
}
