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
	mc := &mockClaude{response: "Summary of notes."}
	e := NewExecutor(mc, mockClipboard("", errors.New("empty")))

	result, err := e.Execute(context.Background(), "summarize", "the meeting notes", "the meeting notes")
	if err != nil {
		t.Fatal(err)
	}
	if result != "Summary of notes." {
		t.Errorf("got %q", result)
	}
	if mc.lastUser != "the meeting notes" {
		t.Errorf("expected freeform text, got %q", mc.lastUser)
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
	if !strings.Contains(mc.lastSystem, "commit message") {
		t.Error("rewrite should include the target format in prompt")
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
	if !strings.Contains(mc.lastSystem, "Spanish") {
		t.Error("translate should include target language")
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
	e := NewExecutor(mc, mockClipboard("", nil))

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
