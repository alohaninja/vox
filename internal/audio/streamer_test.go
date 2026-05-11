package audio

import (
	"context"
	"testing"
	"time"
)

func TestNewStreamer(t *testing.T) {
	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool available: %v", err)
	}
	if s.SampleRate != defaultSampleRate {
		t.Errorf("SampleRate = %d, want %d", s.SampleRate, defaultSampleRate)
	}
	if s.ChunkInterval != defaultChunkInterval {
		t.Errorf("ChunkInterval = %v, want %v", s.ChunkInterval, defaultChunkInterval)
	}
}

func TestStreamerStopWithoutStart(t *testing.T) {
	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}
	_, err = s.Stop()
	if err != ErrNotRecording {
		t.Errorf("Stop without Start should return ErrNotRecording, got %v", err)
	}
}

func TestStreamerIsStreaming(t *testing.T) {
	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}
	if s.IsStreaming() {
		t.Error("should not be streaming before Start")
	}
}

func TestStreamerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}
	s.ChunkInterval = 500 * time.Millisecond

	transcribeCalls := 0
	mockTranscribe := func(_ context.Context, _ []byte) (string, error) {
		transcribeCalls++
		return "partial text", nil
	}

	textCh, err := s.Start(context.Background(), mockTranscribe)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !s.IsStreaming() {
		t.Error("should be streaming after Start")
	}

	// Let it record briefly.
	time.Sleep(1500 * time.Millisecond)

	_, err = s.Stop()
	// May get ErrTooShort for very brief recordings -- that's fine.
	if err != nil && err != ErrTooShort {
		t.Fatalf("Stop: %v", err)
	}

	if s.IsStreaming() {
		t.Error("should not be streaming after Stop")
	}

	// Drain the channel.
	for range textCh {
	}
}

func TestStreamerDoubleStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}

	noop := func(_ context.Context, _ []byte) (string, error) { return "", nil }
	_, err = s.Start(context.Background(), noop)
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	defer s.Stop()

	_, err = s.Start(context.Background(), noop)
	if err == nil {
		t.Fatal("second Start should error")
	}
}
