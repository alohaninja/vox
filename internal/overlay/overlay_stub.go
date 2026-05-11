//go:build !darwin

package overlay

import "fmt"

// StubOverlay prints transcription to stderr on non-darwin platforms.
type StubOverlay struct{}

// NewStubOverlay creates a stub overlay.
func NewStubOverlay() *StubOverlay { return &StubOverlay{} }

func (o *StubOverlay) Show()                  {}
func (o *StubOverlay) Hide()                  {}
func (o *StubOverlay) UpdateText(text string) { fmt.Printf("[overlay] %s\n", text) }
func (o *StubOverlay) SetStatus(s Status)     { fmt.Printf("[overlay] %s\n", s) }
