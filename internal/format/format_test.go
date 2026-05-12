package format

import (
	"strings"
	"testing"

	"vox/internal/appctx"
)

func TestHintByCategory(t *testing.T) {
	tests := []struct {
		category appctx.Category
		contains string // expected substring in the hint
		empty    bool   // whether hint should be empty
	}{
		{appctx.CategoryTerminal, "terminal", false},
		{appctx.CategoryIDE, "code editor", false},
		{appctx.CategoryChat, "chat", false},
		{appctx.CategoryBrowser, "browser", false},
		{appctx.CategoryEditor, "text editor", false},
		{appctx.CategoryUnknown, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.category.String(), func(t *testing.T) {
			hint := Hint(tt.category)
			if tt.empty {
				if hint != "" {
					t.Errorf("expected empty hint for %s, got %q", tt.category, hint)
				}
				return
			}
			if hint == "" {
				t.Errorf("expected non-empty hint for %s", tt.category)
				return
			}
			if !strings.Contains(strings.ToLower(hint), tt.contains) {
				t.Errorf("hint for %s should contain %q, got %q", tt.category, tt.contains, hint)
			}
		})
	}
}

func TestSystemPromptWithHint(t *testing.T) {
	base := "Fix grammar and punctuation."

	// Known category: hint appended.
	result := SystemPromptWithHint(base, appctx.CategoryTerminal)
	if !strings.HasPrefix(result, base) {
		t.Error("should start with base prompt")
	}
	if !strings.Contains(result, "terminal") {
		t.Error("should contain terminal hint")
	}

	// Unknown category: base prompt unchanged.
	result = SystemPromptWithHint(base, appctx.CategoryUnknown)
	if result != base {
		t.Errorf("unknown category should return base prompt unchanged, got %q", result)
	}
}

func TestHintNotEmpty(t *testing.T) {
	// Every known category should have a non-empty hint.
	categories := []appctx.Category{
		appctx.CategoryTerminal,
		appctx.CategoryIDE,
		appctx.CategoryChat,
		appctx.CategoryBrowser,
		appctx.CategoryEditor,
	}
	for _, cat := range categories {
		if Hint(cat) == "" {
			t.Errorf("Hint(%s) should not be empty", cat)
		}
	}
}

func TestHintsAreDistinct(t *testing.T) {
	// Each category should produce a unique hint.
	seen := make(map[string]appctx.Category)
	categories := []appctx.Category{
		appctx.CategoryTerminal,
		appctx.CategoryIDE,
		appctx.CategoryChat,
		appctx.CategoryBrowser,
		appctx.CategoryEditor,
	}
	for _, cat := range categories {
		h := Hint(cat)
		if prev, ok := seen[h]; ok {
			t.Errorf("duplicate hint between %s and %s", prev, cat)
		}
		seen[h] = cat
	}
}
