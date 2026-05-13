package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"vox/internal/audio"
	"vox/internal/hotkey"
	"vox/internal/transcribe"
)

// runTest performs a comprehensive self-diagnosis of all Vox subsystems.
// Each check prints [PASS], [FAIL], or [SKIP] so users can quickly see
// what's working and what needs attention.
func runTest() {
	fmt.Print(banner)
	fmt.Println("Self-Test")
	fmt.Println("=========")
	fmt.Println()

	passed, failed, skipped := 0, 0, 0

	check := func(name string, fn func() (string, error)) {
		result, err := fn()
		if err != nil {
			fmt.Printf("  [FAIL] %s: %v\n", name, err)
			failed++
		} else {
			if result != "" {
				fmt.Printf("  [PASS] %s (%s)\n", name, result)
			} else {
				fmt.Printf("  [PASS] %s\n", name)
			}
			passed++
		}
	}

	skip := func(name, reason string) {
		fmt.Printf("  [SKIP] %s: %s\n", name, reason)
		skipped++
	}

	// 1. Permissions
	fmt.Println("[1/4] Permissions")
	check("Accessibility", func() (string, error) {
		if !hotkey.CheckAccessibility() {
			return "", fmt.Errorf("not granted -- System Settings > Privacy > Accessibility")
		}
		return "granted", nil
	})
	check("Microphone", func() (string, error) {
		if !hotkey.RequestMicrophoneAccess() {
			return "", fmt.Errorf("not granted -- System Settings > Privacy > Microphone")
		}
		return "granted", nil
	})
	fmt.Println()

	// 2. Dependencies
	fmt.Println("[2/4] Dependencies")
	check("Recording tool", func() (string, error) {
		if _, err := exec.LookPath("rec"); err == nil {
			return "sox (rec)", nil
		}
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			return "ffmpeg", nil
		}
		return "", fmt.Errorf("neither sox nor ffmpeg found -- brew install sox")
	})
	check("Whisper server", func() (string, error) {
		if _, err := exec.LookPath("whisper-server"); err != nil {
			return "", fmt.Errorf("not found -- brew install whisper-cpp")
		}
		return "installed", nil
	})
	fmt.Println()

	// 3. Connectivity
	fmt.Println("[3/4] Connectivity")
	whisperURL := os.Getenv("WHISPER_URL")
	if whisperURL == "" {
		whisperURL = "http://127.0.0.1:2022"
	}
	client := transcribe.NewClient(whisperURL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	whisperOK := false
	check("Whisper server reachable", func() (string, error) {
		if err := client.HealthCheck(ctx); err != nil {
			return "", fmt.Errorf("%s unreachable -- is whisper-server running? (make start)", whisperURL)
		}
		whisperOK = true
		return whisperURL, nil
	})

	// 4. Recording + transcription (only if whisper is up)
	fmt.Println()
	fmt.Println("[4/4] Recording & Transcription")
	if !whisperOK {
		skip("Test recording", "whisper server not reachable")
		skip("Transcription", "whisper server not reachable")
	} else {
		recorder, err := audio.NewRecorder()
		if err != nil {
			skip("Test recording", fmt.Sprintf("no recorder: %v", err))
			skip("Transcription", "no recorder")
		} else {
			check("Test recording (1s)", func() (string, error) {
				if err := recorder.Start(); err != nil {
					return "", fmt.Errorf("start failed: %w", err)
				}
				time.Sleep(1200 * time.Millisecond)
				data, err := recorder.Stop()
				if err != nil {
					return "", fmt.Errorf("stop failed: %w", err)
				}
				return fmt.Sprintf("%d bytes", len(data)), nil
			})

			// Quick transcription test with a fresh 1s recording.
			check("Transcription", func() (string, error) {
				if err := recorder.Start(); err != nil {
					return "", fmt.Errorf("start failed: %w", err)
				}
				time.Sleep(1200 * time.Millisecond)
				data, err := recorder.Stop()
				if err != nil {
					return "", fmt.Errorf("stop failed: %w", err)
				}
				text, err := client.Transcribe(ctx, data, transcribe.TranscribeOptions{})
				if err != nil {
					return "", fmt.Errorf("transcribe failed: %w", err)
				}
				if text == "" {
					return "(silence detected)", nil
				}
				return fmt.Sprintf("%q", text), nil
			})
		}
	}

	// 5. AI Features
	fmt.Println()
	fmt.Println("[5/5] AI Features")
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		check("ANTHROPIC_API_KEY", func() (string, error) {
			return "set", nil
		})
	} else {
		skip("ANTHROPIC_API_KEY", "not set -- AI features disabled")
	}
	ldKey := os.Getenv("VOX_LD_SDK_KEY")
	if ldKey != "" {
		check("VOX_LD_SDK_KEY", func() (string, error) {
			return "set", nil
		})
	} else {
		skip("VOX_LD_SDK_KEY", "not set -- using local config only")
	}

	// Summary
	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed, %d skipped\n", passed, failed, skipped)
	if failed > 0 {
		fmt.Println("\nRun 'vox help' for troubleshooting tips.")
		os.Exit(1)
	}
}
