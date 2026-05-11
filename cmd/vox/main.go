package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.design/x/mainthread"

	"vox/internal/audio"
	"vox/internal/config"
	"vox/internal/hotkey"
	"vox/internal/inject"
	"vox/internal/pipeline"
	"vox/internal/transcribe"
)

const banner = `
 __   _____ _  __
 \ \ / / _ \ \/ /
  \ V / (_) >  <
   \_/ \___/_/\_\
  speech-to-text
`

func main() {
	mainthread.Init(run)
}

func run() {
	log.SetFlags(0)

	if len(os.Args) > 1 && os.Args[1] == "setup" {
		runSetup()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	fmt.Print(banner)
	fmt.Printf("Config: %s\n\n", cfg)

	var logger *slog.Logger
	if cfg.Verbose {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Check Accessibility permission.
	if !hotkey.CheckAccessibility() {
		fmt.Fprintln(os.Stderr, "Error: Accessibility permission required.")
		fmt.Fprintln(os.Stderr, "  Grant it in: System Settings > Privacy & Security > Accessibility")
		fmt.Fprintln(os.Stderr, "  Add your terminal app (Terminal, iTerm2, etc.) to the list.")
		os.Exit(1)
	}

	// Check Microphone permission.
	if !hotkey.RequestMicrophoneAccess() {
		fmt.Fprintln(os.Stderr, "❌ Microphone denied — grant it in System Settings > Privacy & Security > Microphone")
		os.Exit(1)
	}

	// Check Whisper server.
	client := transcribe.NewClient(cfg.WhisperURL)
	if err := client.HealthCheck(ctx); err != nil {
		fmt.Printf("Warning: Whisper server unavailable at %s (%v)\n", cfg.WhisperURL, err)
		fmt.Println("  Transcription will fail until the server is reachable.")
		fmt.Println()
	} else if cfg.Verbose {
		logger.Debug("whisper health check passed", "url", cfg.WhisperURL)
	}

	// Clean up any orphaned temp files from prior crashes in the background.
	go func() {
		if n := audio.CleanupOrphanedTempFiles(); n > 0 {
			logger.Info("cleaned up orphaned temp files", "count", n)
		}
	}()

	// Create audio recorder.
	recorder, err := audio.NewRecorder()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Install sox: brew install sox")
		os.Exit(1)
	}

	// Build hotkey labels for display.
	var labels []string
	for _, t := range cfg.Triggers {
		labels = append(labels, t.Label)
	}
	hotkeyLabel := strings.Join(labels, " or ")

	// Create hotkey listener.
	listener := hotkey.NewListener(cfg.Triggers)

	if cfg.HoldToTalk {
		fmt.Printf("✅ Vox ready — Hold %s to dictate.\n", hotkeyLabel)
	} else {
		fmt.Printf("✅ Vox ready — Press %s to start/stop dictation.\n", hotkeyLabel)
	}
	fmt.Println()

	// Build the processing pipeline.
	pipe := pipeline.New(
		transcribeStage(client),
		filterBlankStage(),
		injectStage(),
	)

	// Signal handling.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run event loop on a separate goroutine (listener.Start blocks the main thread).
	if cfg.HoldToTalk {
		go runHoldToTalk(ctx, cfg, logger, listener, recorder, pipe, sigCh)
	} else {
		go runToggle(ctx, cfg, logger, listener, recorder, pipe, sigCh)
	}

	// Start CGEventTap on the main thread (blocks forever).
	if err := listener.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSetup() {
	fmt.Println("Checking permissions...")
	fmt.Println()

	fmt.Print("Accessibility: ")
	if hotkey.CheckAccessibility() {
		fmt.Println("✅ granted")
	} else {
		fmt.Println("⏳ requested — grant it in System Settings > Privacy & Security > Accessibility")
	}

	fmt.Print("Microphone:    ")
	if hotkey.RequestMicrophoneAccess() {
		fmt.Println("✅ granted")
	} else {
		fmt.Println("❌ denied — grant it in System Settings > Privacy & Security > Microphone")
	}

	fmt.Println()
	fmt.Println("Done. Run: make start")
}

func runHoldToTalk(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	listener *hotkey.Listener,
	recorder *audio.Recorder,
	pipe *pipeline.Pipeline,
	sigCh <-chan os.Signal,
) {
	for {
		select {
		case <-sigCh:
			cleanup(logger, recorder)
			os.Exit(0)
		case <-listener.Keydown():
			if cfg.Verbose {
				logger.Debug("hotkey pressed, starting recording")
			}
			audio.PlayStartSound()
			fmt.Println("Recording...")
			if err := recorder.Start(); err != nil {
				fmt.Printf("Error starting recording: %v\n", err)
				continue
			}
		case <-listener.Keyup():
			if !recorder.IsRecording() {
				continue
			}
			audio.PlayStopSound()
			handleStopAndProcess(ctx, cfg, logger, recorder, pipe)
		}
	}
}

func runToggle(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	listener *hotkey.Listener,
	recorder *audio.Recorder,
	pipe *pipeline.Pipeline,
	sigCh <-chan os.Signal,
) {
	for {
		select {
		case <-sigCh:
			cleanup(logger, recorder)
			os.Exit(0)
		case <-listener.Keydown():
			if recorder.IsRecording() {
				audio.PlayStopSound()
				handleStopAndProcess(ctx, cfg, logger, recorder, pipe)
			} else {
				if cfg.Verbose {
					logger.Debug("hotkey pressed, starting recording (toggle mode)")
				}
				audio.PlayStartSound()
				fmt.Println("Recording... (press again to stop)")
				if err := recorder.Start(); err != nil {
					fmt.Printf("Error starting recording: %v\n", err)
				}
			}
		case <-listener.Keyup():
			// In toggle mode, keyup is ignored.
		}
	}
}

func handleStopAndProcess(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	recorder *audio.Recorder,
	pipe *pipeline.Pipeline,
) {
	fmt.Println("Transcribing...")

	wavData, err := recorder.Stop()
	if err != nil {
		if errors.Is(err, audio.ErrTooShort) {
			fmt.Println("Too short, skipping.")
			fmt.Println("Ready!")
			return
		}
		fmt.Printf("Error stopping recording: %v\n", err)
		fmt.Println("Ready!")
		return
	}

	if cfg.Verbose {
		logger.Debug("recorded audio", "bytes", len(wavData))
	}

	r := &pipeline.Result{RawAudio: wavData}
	if err := pipe.Run(ctx, r); err != nil {
		fmt.Printf("Error processing: %v\n", err)
		fmt.Println("Ready!")
		return
	}

	if r.Cancelled {
		fmt.Println("(no speech detected)")
	}

	fmt.Println("Ready!")
}

// transcribeStage returns a pipeline stage that sends audio to the Whisper API.
func transcribeStage(client *transcribe.Client) pipeline.Stage {
	return func(ctx context.Context, r *pipeline.Result) error {
		text, err := client.Transcribe(ctx, r.RawAudio)
		if err != nil {
			return err
		}
		r.RawText = text
		r.OutputText = text
		return nil
	}
}

// filterBlankStage returns a pipeline stage that cancels processing when the
// transcription is blank or a whisper hallucination artifact.
func filterBlankStage() pipeline.Stage {
	return func(_ context.Context, r *pipeline.Result) error {
		if r.RawText == "" || isBlankAudio(r.RawText) {
			r.Cancelled = true
		}
		return nil
	}
}

// injectStage returns a pipeline stage that pastes the output text into the
// focused application via the system clipboard.
func injectStage() pipeline.Stage {
	return func(_ context.Context, r *pipeline.Result) error {
		fmt.Printf(">>> %s\n", r.OutputText)
		return inject.TypeText(r.OutputText)
	}
}

// isBlankAudio returns true if the transcription is a whisper hallucination
// artifact rather than real speech (e.g. "[BLANK_AUDIO]", "[blank audio]", "(blank audio)").
func isBlankAudio(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	t = strings.Trim(t, "[]() ")
	return t == "blank audio" || t == "blank_audio"
}

func cleanup(logger *slog.Logger, recorder *audio.Recorder) {
	if recorder.IsRecording() {
		if _, err := recorder.Stop(); err != nil && !errors.Is(err, audio.ErrNotRecording) {
			logger.Warn("error stopping recorder during shutdown", "error", err)
		}
	}
	fmt.Println("\nStopped.")
}
