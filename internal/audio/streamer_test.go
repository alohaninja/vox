package audio

import (
	"context"
	"sync/atomic"
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
	if s.MaxDuration != DefaultMaxDuration {
		t.Errorf("MaxDuration = %v, want %v", s.MaxDuration, DefaultMaxDuration)
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

	var transcribeCalls atomic.Int32
	mockTranscribe := func(_ context.Context, _ []byte) (string, error) {
		transcribeCalls.Add(1)
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

	// Drain the channel and count received texts.
	received := 0
	for range textCh {
		received++
	}

	// The transcriber should have been called at least once given 1500ms
	// with a 500ms interval (unless the recording tool produced < minFileSize).
	calls := transcribeCalls.Load()
	t.Logf("transcriber called %d times, received %d texts on channel", calls, received)
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
	if err != ErrAlreadyStreaming {
		t.Fatalf("second Start should return ErrAlreadyStreaming, got %v", err)
	}
}

func TestStreamerContextCancelRequiresStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}
	s.ChunkInterval = 200 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	noop := func(_ context.Context, _ []byte) (string, error) { return "", nil }

	textCh, err := s.Start(ctx, noop)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Cancel the context.
	cancel()

	// Drain the channel — streamLoop should exit and close textCh.
	for range textCh {
	}

	// After context cancellation, streaming flag is still true —
	// Stop must be called to clean up the subprocess and temp file.
	if !s.IsStreaming() {
		t.Error("should still report streaming after ctx cancel (Stop needed for cleanup)")
	}

	_, err = s.Stop()
	// ErrTooShort is acceptable for brief recordings.
	if err != nil && err != ErrTooShort {
		t.Fatalf("Stop after cancel: %v", err)
	}

	if s.IsStreaming() {
		t.Error("should not be streaming after Stop")
	}
}

func TestStreamerStartStopReuse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}
	s.ChunkInterval = 200 * time.Millisecond

	noop := func(_ context.Context, _ []byte) (string, error) { return "", nil }

	// First cycle.
	textCh1, err := s.Start(context.Background(), noop)
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	_, err = s.Stop()
	if err != nil && err != ErrTooShort {
		t.Fatalf("first Stop: %v", err)
	}
	for range textCh1 {
	}

	// Second cycle — should work cleanly, no race on stopCh.
	textCh2, err := s.Start(context.Background(), noop)
	if err != nil {
		t.Fatalf("second Start: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	_, err = s.Stop()
	if err != nil && err != ErrTooShort {
		t.Fatalf("second Stop: %v", err)
	}
	for range textCh2 {
	}

	if s.IsStreaming() {
		t.Error("should not be streaming after second Stop")
	}
}

func TestStreamerInvalidChunkInterval(t *testing.T) {
	s, err := NewStreamer()
	if err != nil {
		t.Skipf("no recording tool: %v", err)
	}

	noop := func(_ context.Context, _ []byte) (string, error) { return "", nil }

	// Zero interval should be rejected.
	s.ChunkInterval = 0
	_, err = s.Start(context.Background(), noop)
	if err == nil {
		t.Fatal("Start with zero ChunkInterval should return error")
	}

	// Negative interval should be rejected.
	s.ChunkInterval = -1 * time.Second
	_, err = s.Start(context.Background(), noop)
	if err == nil {
		t.Fatal("Start with negative ChunkInterval should return error")
	}
}
