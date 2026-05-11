//go:build !darwin

package overlay

import (
	"fmt"
	"os"
)

// Compile-time interface check.
var _ Overlay = (*StubOverlay)(nil)

// StubOverlay prints transcription to stderr on non-darwin platforms.
type StubOverlay struct{}

// New creates a stub overlay for non-darwin platforms.
func New() *StubOverlay { return &StubOverlay{} }

func (o *StubOverlay) Show()  {}
func (o *StubOverlay) Hide()  {}
func (o *StubOverlay) Close() {}

func (o *StubOverlay) UpdateText(text string) {
	fmt.Fprintf(os.Stderr, "[overlay] %s\n", text)
}

func (o *StubOverlay) SetStatus(s Status) {
	fmt.Fprintf(os.Stderr, "[overlay] %s\n", s)
}
