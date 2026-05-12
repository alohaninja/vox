//go:build !darwin

package hotkey

// RequestMicrophoneAccess always returns true on non-darwin platforms
// (no OS-level permission prompt needed).
func RequestMicrophoneAccess() bool { return true }
