package inject

import "errors"

// ErrNotSupported is returned by stubs on non-darwin platforms where the
// operation compiles but has no real implementation yet.
var ErrNotSupported = errors.New("not supported on this platform")

// ErrPasteKeyUnknown is returned when the active keyboard layout cannot be read
// to find the key that produces 'v'. No paste keystroke is posted and the text
// is left on the clipboard for the user to paste by hand.
var ErrPasteKeyUnknown = errors.New("cannot find the paste key for the active keyboard layout; text left on the clipboard")
