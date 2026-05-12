//go:build darwin

package overlay

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>

void overlayShow(void);
void overlayHide(void);
void overlayUpdateText(const char *text);
void overlaySetStatus(const char *status);
void overlayInit(void);
void overlayClose(void);
*/
import "C"

import (
	"sync"
	"unicode/utf8"
	"unsafe"

	"golang.design/x/mainthread"
)

// Compile-time interface check.
var _ Overlay = (*DarwinOverlay)(nil)

var (
	darwinOnce     sync.Once
	darwinInstance *DarwinOverlay
	darwinInitDone bool
)

// DarwinOverlay is a macOS floating window overlay for real-time transcription.
// It uses a borderless, always-on-top NSWindow positioned at the bottom-center
// of the screen. All AppKit operations are dispatched to the main thread via
// mainthread.Call to satisfy macOS threading requirements.
//
// DarwinOverlay is a singleton — multiple calls to New() return the same instance.
type DarwinOverlay struct{}

// New creates and initialises the macOS overlay window. It is safe to call
// from any goroutine; the actual NSWindow creation happens synchronously on
// the main thread. Subsequent calls return the same singleton instance.
func New() *DarwinOverlay {
	darwinOnce.Do(func() {
		mainthread.Call(func() {
			C.overlayInit()
		})
		darwinInitDone = true
		darwinInstance = &DarwinOverlay{}
	})
	return darwinInstance
}

func (o *DarwinOverlay) Show() {
	if !darwinInitDone {
		return
	}
	mainthread.Call(func() {
		C.overlayShow()
	})
}

func (o *DarwinOverlay) Hide() {
	if !darwinInitDone {
		return
	}
	mainthread.Call(func() {
		C.overlayHide()
	})
}

func (o *DarwinOverlay) UpdateText(text string) {
	if !darwinInitDone {
		return
	}
	if !utf8.ValidString(text) {
		text = "[invalid text]"
	}
	mainthread.Call(func() {
		cs := C.CString(text)
		defer C.free(unsafe.Pointer(cs))
		C.overlayUpdateText(cs)
	})
}

func (o *DarwinOverlay) SetStatus(status Status) {
	if !darwinInitDone {
		return
	}
	s := status.String()
	mainthread.Call(func() {
		cs := C.CString(s)
		defer C.free(unsafe.Pointer(cs))
		C.overlaySetStatus(cs)
	})
}

func (o *DarwinOverlay) Close() {
	if !darwinInitDone {
		return
	}
	mainthread.Call(func() {
		C.overlayClose()
	})
	darwinInitDone = false
}
