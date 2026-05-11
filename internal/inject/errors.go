package inject

import "errors"

// ErrNotSupported is returned by stubs on non-darwin platforms where the
// operation compiles but has no real implementation yet.
var ErrNotSupported = errors.New("not supported on this platform")
