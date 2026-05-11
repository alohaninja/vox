//go:build darwin

package appctx

import "testing"

func TestCategoryClassification(t *testing.T) {
	tests := []struct {
		name     string
		bundleID string
		want     Category
	}{
		// Terminals
		{"Terminal.app", "com.apple.Terminal", CategoryTerminal},
		{"iTerm2", "com.googlecode.iterm2", CategoryTerminal},
		{"Kitty", "net.kovidgoyal.kitty", CategoryTerminal},
		{"Alacritty", "io.alacritty", CategoryTerminal},
		{"WezTerm", "com.github.wez.wezterm", CategoryTerminal},
		{"Warp", "dev.warp.warp-stable", CategoryTerminal},
		{"Hyper", "co.zeit.hyper", CategoryTerminal},

		// IDEs
		{"VS Code", "com.microsoft.VSCode", CategoryIDE},
		{"VS Code case", "com.microsoft.vscode", CategoryIDE},
		{"IntelliJ", "com.jetbrains.intellij", CategoryIDE},
		{"GoLand", "com.jetbrains.goland", CategoryIDE},
		{"Xcode", "com.apple.dt.Xcode", CategoryIDE},
		{"Zed", "dev.zed.Zed", CategoryIDE},
		{"Cursor", "com.cursor.Cursor", CategoryIDE},
		{"Sublime Text", "com.sublimetext.4", CategoryIDE},

		// Chat
		{"Slack", "com.tinyspeck.slackmacgap", CategoryChat},
		{"Discord legacy", "com.hnc.Discord", CategoryChat},
		{"Discord new", "com.discord.Discord", CategoryChat},
		{"Teams", "com.microsoft.teams", CategoryChat},
		{"Telegram", "ru.keepcoder.Telegram", CategoryChat},
		{"Messenger", "com.facebook.archon", CategoryChat},
		{"Signal", "org.whispersystems.signal-desktop", CategoryChat},
		{"Messages macOS", "com.apple.MobileSMS", CategoryChat},

		// Browsers
		{"Safari", "com.apple.Safari", CategoryBrowser},
		{"Chrome", "com.google.Chrome", CategoryBrowser},
		{"Firefox", "org.mozilla.firefox", CategoryBrowser},
		{"Edge", "com.microsoft.edgemac", CategoryBrowser},
		{"Brave", "com.brave.Browser", CategoryBrowser},
		{"Arc", "company.thebrowser.Browser", CategoryBrowser},

		// Editors
		{"Notes", "com.apple.Notes", CategoryEditor},
		{"TextEdit", "com.apple.TextEdit", CategoryEditor},
		{"Pages", "com.apple.Pages", CategoryEditor},
		{"Pages iWork", "com.apple.iWork.Pages", CategoryEditor},
		{"Obsidian", "md.obsidian", CategoryEditor},
		{"Notion", "com.notion.Notion", CategoryEditor},
		{"Typora", "abnerworks.Typora", CategoryEditor},

		// Unknown
		{"Unknown app", "com.example.unknown", CategoryUnknown},
		{"Empty", "", CategoryUnknown},
		{"Finder", "com.apple.finder", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := AppContext{BundleID: tt.bundleID}
			got := ctx.Category()
			if got != tt.want {
				t.Errorf("Category(%q) = %v, want %v", tt.bundleID, got, tt.want)
			}
		})
	}
}

func TestCategoryString(t *testing.T) {
	tests := []struct {
		cat  Category
		want string
	}{
		{CategoryTerminal, "terminal"},
		{CategoryIDE, "ide"},
		{CategoryChat, "chat"},
		{CategoryBrowser, "browser"},
		{CategoryEditor, "editor"},
		{CategoryUnknown, "unknown"},
		{Category(99), "unknown"}, // out-of-range defaults to unknown
	}
	for _, tt := range tests {
		if got := tt.cat.String(); got != tt.want {
			t.Errorf("Category(%d).String() = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

// TestHasPrefixNoFalsePositives verifies that switching from
// strings.Contains to strings.HasPrefix prevents substring false positives.
func TestHasPrefixNoFalsePositives(t *testing.T) {
	tests := []struct {
		name     string
		bundleID string
		want     Category
	}{
		// These would have been false positives with strings.Contains
		{"not a JetBrains app", "evil.com.jetbrains.analytics", CategoryUnknown},
		{"not Terminal", "someprefix.com.apple.terminal", CategoryUnknown},
		{"not Chrome", "notcom.google.chrome.helper", CategoryUnknown},
		{"not Obsidian", "com.amd.obsidian.tool", CategoryUnknown},
		{"updater not IDE", "notcom.cursor.updater", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := AppContext{BundleID: tt.bundleID}
			got := ctx.Category()
			if got != tt.want {
				t.Errorf("Category(%q) = %v, want %v (false positive)", tt.bundleID, got, tt.want)
			}
		})
	}
}

func TestDetectSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping app detection test in short mode")
	}
	ctx := Detect()
	// Verify it doesn't panic and returns something.
	t.Logf("Detected app: BundleID=%q Name=%q Category=%s", ctx.BundleID, ctx.Name, ctx.Category())
	// BundleID should be non-empty when running on a real macOS desktop.
	if ctx.BundleID == "" {
		t.Log("warning: BundleID is empty (expected on headless CI)")
	}
}
