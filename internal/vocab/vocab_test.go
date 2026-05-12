package vocab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	got, err := Load("", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestLoadTermsOnly(t *testing.T) {
	got, err := Load("flagbearer,streamer,gonfalon", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "The following terms may appear: flagbearer, streamer, gonfalon."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadSingleTerm(t *testing.T) {
	got, err := Load("LaunchDarkly", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "The following terms may appear: LaunchDarkly."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadTermsWithWhitespace(t *testing.T) {
	got, err := Load("  flagbearer , streamer , gonfalon  ", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "The following terms may appear: flagbearer, streamer, gonfalon."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadFileOnly(t *testing.T) {
	f := writeTemp(t, "flagbearer\nstreamer\ngonfalon\n")
	got, err := Load("", f)
	if err != nil {
		t.Fatal(err)
	}
	want := "The following terms may appear: flagbearer, streamer, gonfalon."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadFileWithCommentsAndBlanks(t *testing.T) {
	f := writeTemp(t, "# Domain terms\nflagbearer\n\n# More terms\nstreamer\n\n")
	got, err := Load("", f)
	if err != nil {
		t.Fatal(err)
	}
	want := "The following terms may appear: flagbearer, streamer."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadBothTermsAndFile(t *testing.T) {
	f := writeTemp(t, "fdcore\nLaunchDarkly\n")
	got, err := Load("flagbearer,streamer", f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "flagbearer") || !strings.Contains(got, "fdcore") {
		t.Fatalf("expected both env and file terms, got %q", got)
	}
}

func TestLoadDeduplicatesTerms(t *testing.T) {
	f := writeTemp(t, "flagbearer\ngonfalon\n")
	got, err := Load("flagbearer,Flagbearer,gonfalon", f)
	if err != nil {
		t.Fatal(err)
	}
	want := "The following terms may appear: flagbearer, gonfalon."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadMaxTermsCap(t *testing.T) {
	// Build a comma-separated string with MaxTerms+50 unique terms.
	var terms []string
	for i := 0; i < MaxTerms+50; i++ {
		terms = append(terms, fmt.Sprintf("term%d", i))
	}
	got, err := Load(strings.Join(terms, ","), "")
	if err != nil {
		t.Fatal(err)
	}
	// Count the terms in the output by splitting on ", ".
	prefix := "The following terms may appear: "
	body := strings.TrimSuffix(strings.TrimPrefix(got, prefix), ".")
	count := len(strings.Split(body, ", "))
	if count != MaxTerms {
		t.Fatalf("expected %d terms, got %d", MaxTerms, count)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("", "/nonexistent/path/vocab.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	f := writeTemp(t, "")
	got, err := Load("", f)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty for empty file, got %q", got)
	}
}

func TestLoadCommentsOnlyFile(t *testing.T) {
	f := writeTemp(t, "# just a comment\n# another comment\n")
	got, err := Load("", f)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty for comments-only file, got %q", got)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "vocab.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
