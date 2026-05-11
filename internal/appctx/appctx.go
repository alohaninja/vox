package appctx

import "strings"

// Category represents the type of application.
type Category int

const (
	CategoryUnknown  Category = iota
	CategoryTerminal          // Terminal emulators (Terminal.app, iTerm2, Alacritty, etc.)
	CategoryIDE               // Code editors and IDEs (VS Code, IntelliJ, Xcode, etc.)
	CategoryChat              // Chat and messaging apps (Slack, Discord, Teams, etc.)
	CategoryBrowser           // Web browsers (Safari, Chrome, Firefox, etc.)
	CategoryEditor            // Text/document editors (Notes, TextEdit, Obsidian, etc.)
)

// String returns a human-readable label for the category.
func (c Category) String() string {
	switch c {
	case CategoryTerminal:
		return "terminal"
	case CategoryIDE:
		return "ide"
	case CategoryChat:
		return "chat"
	case CategoryBrowser:
		return "browser"
	case CategoryEditor:
		return "editor"
	default:
		return "unknown"
	}
}

// AppContext holds information about the currently focused application.
// Note: BundleID and Name may contain privacy-sensitive information
// (reveals active user activity). Avoid logging or transmitting without consent.
type AppContext struct {
	BundleID string
	Name     string
}

// Category classifies the focused application based on its bundle ID.
// Patterns ending with "." are treated as prefixes (e.g., "com.jetbrains."
// matches all JetBrains products); all others are matched as prefixes too,
// which is correct for reverse-DNS bundle identifiers.
func (a AppContext) Category() Category {
	bid := strings.ToLower(a.BundleID)

	for _, pattern := range terminalBundleIDs {
		if strings.HasPrefix(bid, pattern) {
			return CategoryTerminal
		}
	}
	for _, pattern := range ideBundleIDs {
		if strings.HasPrefix(bid, pattern) {
			return CategoryIDE
		}
	}
	for _, pattern := range chatBundleIDs {
		if strings.HasPrefix(bid, pattern) {
			return CategoryChat
		}
	}
	for _, pattern := range browserBundleIDs {
		if strings.HasPrefix(bid, pattern) {
			return CategoryBrowser
		}
	}
	for _, pattern := range editorBundleIDs {
		if strings.HasPrefix(bid, pattern) {
			return CategoryEditor
		}
	}

	return CategoryUnknown
}

var terminalBundleIDs = []string{
	"com.apple.terminal",
	"com.googlecode.iterm2",
	"net.kovidgoyal.kitty",
	"io.alacritty",
	"com.github.wez.wezterm",
	"dev.warp.warp-stable",
	"co.zeit.hyper",
}

var ideBundleIDs = []string{
	"com.microsoft.vscode",
	"com.jetbrains.",
	"com.apple.dt.xcode",
	"dev.zed.zed",
	"com.sublimetext.",
	"com.cursor.",
}

var chatBundleIDs = []string{
	"com.tinyspeck.slackmacgap",
	"com.hnc.discord",
	"com.discord.",
	"com.microsoft.teams",
	"ru.keepcoder.telegram",
	"com.facebook.archon",
	"org.whispersystems.signal-desktop",
	"com.apple.mobilesms",
}

var browserBundleIDs = []string{
	"com.apple.safari",
	"com.google.chrome",
	"org.mozilla.firefox",
	"com.microsoft.edgemac",
	"com.brave.browser",
	"company.thebrowser.browser",
}

var editorBundleIDs = []string{
	"com.apple.notes",
	"com.apple.textedit",
	"com.apple.pages",
	"com.apple.iwork.pages",
	"md.obsidian",
	"com.notion.notion",
	"abnerworks.typora",
}
