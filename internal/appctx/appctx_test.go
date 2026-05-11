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

		// IDEs
		{"VS Code", "com.microsoft.VSCode", CategoryIDE},
		{"VS Code case", "com.microsoft.vscode", CategoryIDE},
		{"IntelliJ", "com.jetbrains.intellij", CategoryIDE},
		{"GoLand", "com.jetbrains.goland", CategoryIDE},
		{"Xcode", "com.apple.dt.Xcode", CategoryIDE},
		{"Zed", "dev.zed.Zed", CategoryIDE},
		{"Cursor", "com.cursor.Cursor", CategoryIDE},

		// Chat
		{"Slack", "com.tinyspeck.slackmacgap", CategoryChat},
		{"Discord", "com.hnc.Discord", CategoryChat},
		{"Teams", "com.microsoft.teams", CategoryChat},
		{"Messages", "com.apple.MobileSMS", CategoryUnknown}, // MobileSMS is iOS
		{"Messages mac", "com.apple.messages", CategoryChat},

		// Browsers
		{"Safari", "com.apple.Safari", CategoryBrowser},
		{"Chrome", "com.google.Chrome", CategoryBrowser},
		{"Firefox", "org.mozilla.firefox", CategoryBrowser},
		{"Arc", "company.thebrowser.Browser", CategoryBrowser},

		// Editors
		{"Notes", "com.apple.Notes", CategoryEditor},
		{"TextEdit", "com.apple.TextEdit", CategoryEditor},
		{"Obsidian", "md.obsidian", CategoryEditor},
		{"Notion", "com.notion.Notion", CategoryEditor},

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
	}
	for _, tt := range tests {
		if got := tt.cat.String(); got != tt.want {
			t.Errorf("Category(%d).String() = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

func TestDetectSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping app detection test in short mode")
	}
	ctx := Detect()
	// We just verify it doesn't panic and returns something.
	t.Logf("Detected app: BundleID=%q Name=%q Category=%s", ctx.BundleID, ctx.Name, ctx.Category())
}
