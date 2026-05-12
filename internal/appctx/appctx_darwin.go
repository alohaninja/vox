//go:build darwin

package appctx

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>

const char* getFrontmostAppBundleID(void);
const char* getFrontmostAppName(void);
*/
import "C"

import "unsafe"

// Detect returns the currently focused application on macOS.
func Detect() AppContext {
	cbid := C.getFrontmostAppBundleID()
	cname := C.getFrontmostAppName()

	var bid, name string
	if cbid != nil {
		bid = C.GoString(cbid)
		C.free(unsafe.Pointer(cbid))
	}
	if cname != nil {
		name = C.GoString(cname)
		C.free(unsafe.Pointer(cname))
	}

	return AppContext{BundleID: bid, Name: name}
}
