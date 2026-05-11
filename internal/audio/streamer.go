package audio

import (
	"context"
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

// Transcriber is called with accumulated audio data and returns partial text.
type Transcriber func(ctx context.Context, wavData []byte) (string, error)

// Streamer records audio and periodically sends accumulated chunks to a
// transcriber for incremental (sliding-window) transcription. This enables
// real-time text display while the user is still speaking.
type Streamer struct {
	SampleRate    int
	ChunkInterval time.Duration

	mu        sync.Mutex
	tool      toolKind
	cmd       *exec.Cmd
	tmpPath   string
	streaming bool
	stopCh    chan struct{}
}

// NewStreamer creates a Streamer with the same tool detection as Recorder.
func NewStreamer() (*Streamer, error) {
	s := &Streamer{
		SampleRate:    defaultSampleRate,
		ChunkInterval: defaultChunkInterval,
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
func (s *Streamer) Start(ctx context.Context, transcribe Transcriber) (<-chan string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.streaming {
		return nil, fmt.Errorf("already streaming")
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
	}
	s.cmd.Stdout = nil
	s.cmd.Stderr = nil

	if err := s.cmd.Start(); err != nil {
		_ = os.Remove(s.tmpPath)
		return nil, fmt.Errorf("start recording: %w", err)
	}

	s.streaming = true
	s.stopCh = make(chan struct{})

	textCh := make(chan string, 16)
	go s.streamLoop(ctx, transcribe, textCh)

	return textCh, nil
}

// Stop ends the streaming session and returns the final accumulated audio.
func (s *Streamer) Stop() ([]byte, error) {
	s.mu.Lock()
	if !s.streaming {
		s.mu.Unlock()
		return nil, ErrNotRecording
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
	case <-waitDone:
	case <-time.After(killGracePeriod):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-waitDone
	}

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

func (s *Streamer) streamLoop(ctx context.Context, transcribe Transcriber, textCh chan<- string) {
	defer close(textCh)

	ticker := time.NewTicker(s.ChunkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
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
