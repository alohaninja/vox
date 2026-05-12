//go:build !darwin

package inject

import (
	"fmt"
	"os"
)

// TypeText is a stub on non-darwin platforms. Text injection requires
// platform-specific APIs (xdotool on Linux, SendKeys on Windows).
// The transcribed text is NOT injected into any application; the caller
// is responsible for printing it if desired.
func TypeText(text string) error {
	if text == "" {
		return nil
	}
	return fmt.Errorf("inject text: %w", ErrNotSupported)
}

// ReadClipboard is a stub on non-darwin platforms.
func ReadClipboard() (string, error) {
	return "", fmt.Errorf("read clipboard: %w", ErrNotSupported)
}

// ClearClipboard is a stub on non-darwin platforms.
func ClearClipboard() error {
	return fmt.Errorf("clear clipboard: %w", ErrNotSupported)
}

func init() {
	fmt.Fprintln(os.Stderr, "Note: text injection and clipboard operations are not yet supported on this platform.")
}
