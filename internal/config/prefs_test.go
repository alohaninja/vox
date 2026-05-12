package config

import (
	"path/filepath"
	"sync"
	"testing"
)

// TestSavePref_ConcurrentMutationsPreserveBothFields asserts that concurrent
// SavePref calls from different goroutines (the hotkey watcher and the
// settings watcher in main.go) don't clobber each other's fields. Without
// the mutex inside SavePref this test is racy: each goroutine reads the
// same empty Prefs, then both write back their single change, and one of
// the two updates is lost.
func TestSavePref_ConcurrentMutationsPreserveBothFields(t *testing.T) {
	t.Setenv("VOX_PREFS_PATH", filepath.Join(t.TempDir(), "prefs.json"))

	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if err := SavePref(func(p *Prefs) { p.Hotkey = "fn" }); err != nil {
				t.Errorf("SavePref hotkey: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if err := SavePref(func(p *Prefs) { p.SoundsEnabled = BoolPtr(true) }); err != nil {
				t.Errorf("SavePref sounds: %v", err)
				return
			}
		}
	}()

	wg.Wait()

	got, err := LoadPrefs()
	if err != nil {
		t.Fatalf("LoadPrefs: %v", err)
	}
	if got.Hotkey != "fn" {
		t.Errorf("Hotkey lost across concurrent saves: got %q, want %q", got.Hotkey, "fn")
	}
	if got.SoundsEnabled == nil || !*got.SoundsEnabled {
		t.Errorf("SoundsEnabled lost across concurrent saves: got %v, want true", got.SoundsEnabled)
	}
}
