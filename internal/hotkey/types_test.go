package hotkey

import "testing"

func TestFlagConstants(t *testing.T) {
	// Verify flag constants are non-zero and distinct.
	flags := map[string]uint64{
		"FlagShift":   FlagShift,
		"FlagControl": FlagControl,
		"FlagOption":  FlagOption,
		"FlagCommand": FlagCommand,
		"FlagFn":      FlagFn,
	}
	seen := make(map[uint64]string)
	for name, val := range flags {
		if val == 0 {
			t.Errorf("%s should not be zero", name)
		}
		if prev, ok := seen[val]; ok {
			t.Errorf("%s and %s have the same value: %d", name, prev, val)
		}
		seen[val] = name
	}
}

func TestTriggerDefaults(t *testing.T) {
	var trig Trigger
	if trig.Modifiers != 0 || trig.Key != 0 || trig.Fn || trig.Label != "" {
		t.Error("zero-value Trigger should have all zero fields")
	}
}
