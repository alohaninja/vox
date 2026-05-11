package hotkey

import "testing"

func TestFlagConstants(t *testing.T) {
	// Pin exact macOS CGEvent modifier bitmask values.
	tests := []struct {
		name string
		got  uint64
		want uint64
	}{
		{"FlagShift", FlagShift, 0x20000},
		{"FlagControl", FlagControl, 0x40000},
		{"FlagOption", FlagOption, 0x80000},
		{"FlagCommand", FlagCommand, 0x100000},
		{"FlagFn", FlagFn, 0x800000},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = 0x%x, want 0x%x", tt.name, tt.got, tt.want)
		}
	}

	// Also verify all flags are distinct.
	seen := make(map[uint64]string)
	for _, tt := range tests {
		if prev, ok := seen[tt.got]; ok {
			t.Errorf("%s and %s have the same value: 0x%x", tt.name, prev, tt.got)
		}
		seen[tt.got] = tt.name
	}
}

func TestTriggerDefaults(t *testing.T) {
	var trig Trigger
	if trig.Modifiers != 0 || trig.Key != 0 || trig.Fn || trig.Label != "" {
		t.Error("zero-value Trigger should have all zero fields")
	}
}
