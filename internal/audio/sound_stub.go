//go:build !darwin

package audio

// PlayStartSound is a no-op on non-darwin platforms.
// Sound playback requires platform-specific APIs (paplay on Linux, etc.).
func PlayStartSound() {}

// PlayStopSound is a no-op on non-darwin platforms.
func PlayStopSound() {}
