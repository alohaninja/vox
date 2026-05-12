package hotkey

// CGEvent modifier flag bitmasks (macOS values, used for hotkey parsing on all platforms).
const (
	FlagShift   uint64 = 0x20000
	FlagControl uint64 = 0x40000
	FlagOption  uint64 = 0x80000
	FlagCommand uint64 = 0x100000
	FlagFn      uint64 = 0x800000
)

// Trigger defines a hotkey trigger condition.
type Trigger struct {
	// Modifiers is a bitmask of required modifier flags (FlagShift, FlagCommand, etc.).
	Modifiers uint64
	// Key is a macOS virtual keycode, or -1 for modifier-only triggers.
	Key int
	// Fn triggers on the Fn/Globe key.
	Fn bool
	// Label is a human-readable description (e.g. "Cmd+Shift", "Fn").
	Label string
}
