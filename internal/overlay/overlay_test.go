package overlay

import "testing"

func TestStatusString(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusListening, "Listening..."},
		{StatusTranscribing, "Transcribing..."},
		{StatusReady, "Ready"},
		{Status(99), ""},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusListening != 0 || StatusTranscribing != 1 || StatusReady != 2 {
		t.Error("status constants have unexpected values")
	}
}

func TestOverlayInterfaceCompliance(t *testing.T) {
	// Verify the Overlay interface is satisfiable.
	// On darwin this would test DarwinOverlay, on other platforms StubOverlay.
	// We just verify compilation here.
	var _ Overlay = (*stubForTest)(nil)
}

type stubForTest struct{}

func (s *stubForTest) Show()                   {}
func (s *stubForTest) Hide()                   {}
func (s *stubForTest) UpdateText(text string)  {}
func (s *stubForTest) SetStatus(status Status) {}
