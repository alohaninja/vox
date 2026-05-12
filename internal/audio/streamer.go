package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	defaultChunkInterval = 2 * time.Second
	streamTempPattern    = "vox-stream-*.wav"
)

// ErrAlreadyStreaming is returned when Start is called while already streaming.
var ErrAlreadyStreaming = errors.New("already streaming")

// Transcriber is called with accumulated audio data and returns partial text.
type Transcriber func(ctx context.Context, wavData []byte) (string, error)

// Streamer records audio and periodically sends accumulated chunks to a
// transcriber for incremental (sliding-window) transcription. This enables
// real-time text display while the user is still speaking.
type Streamer struct {
	SampleRate    int
	ChunkInterval time.Duration
	MaxDuration   time.Duration // 0 means no limit; safety cap on streaming duration

	mu        sync.Mutex
	tool      toolKind
	cmd       *exec.Cmd
	tmpPath   string
	streaming bool
	stopCh    chan struct{}
	wg        sync.WaitGroup // tracks the streamLoop goroutine
	timer     *time.Timer    // auto-stop timer
}

// NewStreamer creates a Streamer with the same tool detection as Recorder.
func NewStreamer() (*Streamer, error) {
	s := &Streamer{
		SampleRate:    defaultSampleRate,
		ChunkInterval: defaultChunkInterval,
		MaxDuration:   DefaultMaxDuration,
	}

	if _, err := exec.LookPath("rec"); err == nil {
		s.tool = toolSox
		return s, nil
	}
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		s.tool = toolFFmpeg
		return s, nil
	}
	return nil, fmt.Errorf("no recording tool found: install sox (rec) or ffmpeg")
}

// Start begins streaming audio. It launches a recording subprocess and
// periodically reads the accumulated WAV file, sending chunks to the
// transcriber. Partial transcription results are sent on the returned channel.
//
// Call Stop() to end the stream. The final complete audio is returned by Stop.
// Stop must also be called after context cancellation to clean up the
// subprocess and temp file.
func (s *Streamer) Start(ctx context.Context, transcribe Transcriber) (<-chan string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.streaming {
		return nil, ErrAlreadyStreaming
	}

	if s.ChunkInterval <= 0 {
		return nil, fmt.Errorf("ChunkInterval must be positive, got %v", s.ChunkInterval)
	}

	tmpFile, err := os.CreateTemp("", streamTempPattern)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("close temp file: %w", err)
	}
	s.tmpPath = tmpFile.Name()

	sampleRateStr := fmt.Sprintf("%d", s.SampleRate)
	switch s.tool {
	case toolSox:
		s.cmd = exec.Command("rec", "-q", "-r", sampleRateStr, "-c", "1", "-b", "16", "-t", "wav", s.tmpPath)
	case toolFFmpeg:
		s.cmd = exec.Command("ffmpeg", "-f", "avfoundation", "-i", ":default", "-ar", sampleRateStr, "-ac", "1", "-y", s.tmpPath)
	default:
		_ = os.Remove(s.tmpPath)
		s.tmpPath = ""
		return nil, fmt.Errorf("unknown recording tool: %d", s.tool)
	}
	s.cmd.Stdout = nil
	s.cmd.Stderr = nil

	if err := s.cmd.Start(); err != nil {
		_ = os.Remove(s.tmpPath)
		s.cmd = nil
		s.tmpPath = ""
		return nil, fmt.Errorf("start recording: %w", err)
	}

	s.streaming = true
	s.stopCh = make(chan struct{})

	// Start a safety timer that auto-stops streaming after MaxDuration.
	// This prevents unbounded disk growth if the caller forgets to stop.
	if s.MaxDuration > 0 {
		s.timer = time.AfterFunc(s.MaxDuration, func() {
			_, _ = s.Stop()
		})
	}

	textCh := make(chan string, 16)

	// Capture stopCh as a local so the goroutine never reads s.stopCh
	// from the struct field — prevents a race on rapid Stop→Start cycles.
	stopCh := s.stopCh

	s.wg.Add(1)
	go s.streamLoop(ctx, transcribe, textCh, stopCh)

	return textCh, nil
}

// Stop ends the streaming session and returns the final accumulated audio.
func (s *Streamer) Stop() ([]byte, error) {
	s.mu.Lock()
	if !s.streaming {
		s.mu.Unlock()
		return nil, ErrNotRecording
	}

	// Cancel the safety timer if it hasn't fired yet.
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	cmd := s.cmd
	tmpPath := s.tmpPath
	stopCh := s.stopCh

	s.streaming = false
	s.cmd = nil
	s.tmpPath = ""
	s.mu.Unlock()

	// Signal the stream loop to stop.
	close(stopCh)

	// Gracefully stop the recording process.
	if cmd.Process != nil {
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			_ = cmd.Process.Kill()
		}
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	select {
	case waitErr := <-waitDone:
		if waitErr != nil {
			// sox and ffmpeg exit with non-zero status on SIGINT — that is expected.
			// We only propagate errors that are NOT an ExitError (e.g. process not started).
			var exitErr *exec.ExitError
			if !errors.As(waitErr, &exitErr) {
				// Wait for the streamLoop goroutine to finish before returning.
				s.wg.Wait()
				_ = os.Remove(tmpPath)
				return nil, fmt.Errorf("wait for recording process: %w", waitErr)
			}
		}
	case <-time.After(killGracePeriod):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-waitDone
	}

	// Wait for the streamLoop goroutine to finish before reading the file,
	// ensuring textCh is closed before Stop returns.
	s.wg.Wait()

	defer func() { _ = os.Remove(tmpPath) }()

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("read streamed audio: %w", err)
	}
	if len(data) < minFileSize {
		return nil, ErrTooShort
	}
	return data, nil
}

// IsStreaming reports whether a stream is in progress.
func (s *Streamer) IsStreaming() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streaming
}

func (s *Streamer) streamLoop(ctx context.Context, transcribe Transcriber, textCh chan<- string, stopCh <-chan struct{}) {
	defer s.wg.Done()
	defer close(textCh)

	ticker := time.NewTicker(s.ChunkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			tmpPath := s.tmpPath
			streaming := s.streaming
			s.mu.Unlock()

			if !streaming || tmpPath == "" {
				return
			}

			// Read the current accumulated audio (sliding window).
			data, err := os.ReadFile(tmpPath)
			if err != nil || len(data) < minFileSize {
				continue
			}

			text, err := transcribe(ctx, data)
			if err != nil {
				continue
			}
			if text != "" {
				select {
				case textCh <- text:
				default:
					// Drop if consumer is slow.
				}
			}
		}
	}
}
