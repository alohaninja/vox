//go:build darwin

package appctx

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>

typedef struct {
    const char* bundleID;
    const char* name;
} FrontmostApp;

FrontmostApp getFrontmostApp(void);
*/
import "C"

import "unsafe"

// Detect returns the currently focused application on macOS.
// It atomically captures both the bundle ID and name from the same
// NSRunningApplication instance to avoid TOCTOU races.
func Detect() AppContext {
	app := C.getFrontmostApp()

	var bid, name string
	if app.bundleID != nil {
		bid = C.GoString(app.bundleID)
		C.free(unsafe.Pointer(app.bundleID))
	}
	if app.name != nil {
		name = C.GoString(app.name)
		C.free(unsafe.Pointer(app.name))
	}

	return AppContext{BundleID: bid, Name: name}
}
