//go:build !darwin

package overlay

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestNewReturnsOverlay(t *testing.T) {
	o := New()
	if o == nil {
		t.Fatal("New() returned nil")
	}
	var _ Overlay = o
}

func TestStubOverlayOutput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	stub := &StubOverlay{}
	stub.UpdateText("hello")
	stub.SetStatus(StatusReady)

	w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	want := fmt.Sprintf("[overlay] hello\n[overlay] %s\n", StatusReady)
	if got != want {
		t.Errorf("StubOverlay output:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestStubOverlayClose(t *testing.T) {
	stub := &StubOverlay{}
	stub.Close()
}
