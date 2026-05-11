package overlay

// Status represents the current state of the overlay.
type Status int

const (
	StatusListening    Status = iota // Recording in progress
	StatusTranscribing               // Processing audio
	StatusReady                      // Idle, ready for next recording
)

// String returns a human-readable label for the status.
func (s Status) String() string {
	switch s {
	case StatusListening:
		return "Listening..."
	case StatusTranscribing:
		return "Transcribing..."
	case StatusReady:
		return "Ready"
	default:
		return ""
	}
}

// Overlay is the interface for a floating transcription display.
// Implementations are platform-specific (NSWindow on macOS, stub elsewhere).
type Overlay interface {
	// Show makes the overlay visible.
	Show()

	// Hide hides the overlay.
	Hide()

	// UpdateText sets the transcription text displayed in the overlay.
	UpdateText(text string)

	// SetStatus updates the overlay status indicator.
	SetStatus(status Status)

	// Close tears down the overlay and releases resources.
	Close()
}
