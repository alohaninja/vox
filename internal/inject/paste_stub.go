//go:build !darwin

package inject

import "fmt"

// TypeText is a stub on non-darwin platforms. Text injection requires
// platform-specific APIs (xdotool on Linux, SendKeys on Windows).
func TypeText(text string) error {
	if text == "" {
		return nil
	}
	fmt.Printf(">>> %s\n", text)
	fmt.Println("(text injection not yet supported on this platform — printed above)")
	return nil
}

// ReadClipboard is a stub on non-darwin platforms.
func ReadClipboard() (string, error) {
	return "", fmt.Errorf("clipboard reading not yet supported on this platform")
}

// ClearClipboard is a stub on non-darwin platforms.
func ClearClipboard() error {
	return fmt.Errorf("clipboard operations not yet supported on this platform")
}
