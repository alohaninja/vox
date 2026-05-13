package whispermodel

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DefaultID is the model Vox uses when no override exists in env or prefs.
const DefaultID = "base.en"

// Model describes one downloadable whisper.cpp GGML model.
type Model struct {
	ID       string
	Label    string
	Filename string
	URL      string
	Checksum string
	SizeMB   int
}

var catalog = []Model{
	{
		ID:       "tiny.en",
		Label:    "Tiny (English, ~75 MiB)",
		Filename: "ggml-tiny.en.bin",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin",
		Checksum: "c78c86eb1a8faa21b369bcd33207cc90d64ae9df",
		SizeMB:   75,
	},
	{
		ID:       "base.en",
		Label:    "Base (English, ~142 MiB, default)",
		Filename: "ggml-base.en.bin",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
		Checksum: "a03779c86df3323075f5e796cb2ce5029f00ec8869eee3fdfb897afe36c6d002",
		SizeMB:   142,
	},
	{
		ID:       "small.en",
		Label:    "Small (English, ~466 MiB)",
		Filename: "ggml-small.en.bin",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin",
		Checksum: "db8a495a91d927739e50b3fc1cc4c6b8f6c2d022",
		SizeMB:   466,
	},
	{
		ID:       "medium.en",
		Label:    "Medium (English, ~1.5 GiB)",
		Filename: "ggml-medium.en.bin",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.en.bin",
		Checksum: "8c30f0e44ce9560643ebd10bbe50cd20eafd3723",
		SizeMB:   1536,
	},
	{
		ID:       "large-v3-turbo",
		Label:    "Large v3 Turbo (Multilingual, ~1.5 GiB)",
		Filename: "ggml-large-v3-turbo.bin",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin",
		Checksum: "4af2b29d7ec73d781377bfd1758ca957a807e941",
		SizeMB:   1536,
	},
}

// All returns a copy of the supported model catalog.
func All() []Model {
	return slices.Clone(catalog)
}

// ByID resolves a model by stable ID (for prefs/env values).
func ByID(id string) (Model, bool) {
	for _, m := range catalog {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

// ModelDir is the download/cache directory for GGML model files.
// Honors WHISPER_MODEL_DIR for parity with Makefile behavior.
func ModelDir() (string, error) {
	if p := os.Getenv("WHISPER_MODEL_DIR"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "whisper-cpp"), nil
}

// Path returns the target path for the model file under ModelDir.
func Path(m Model) (string, error) {
	dir, err := ModelDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, m.Filename), nil
}

// IsInstalled returns true if the model file exists and is non-empty.
func IsInstalled(m Model) bool {
	path, err := Path(m)
	if err != nil {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0
}

func validateChecksum(gotHex, expected string) error {
	if expected == "" {
		return fmt.Errorf("missing checksum")
	}
	got := strings.ToLower(strings.TrimSpace(gotHex))
	want := strings.ToLower(strings.TrimSpace(expected))
	if got != want {
		return fmt.Errorf("checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}
