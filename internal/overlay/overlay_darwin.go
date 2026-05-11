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
*/
import "C"

import "unsafe"

// DarwinOverlay is a macOS floating window overlay for real-time transcription.
// It uses a borderless, always-on-top NSWindow positioned at the bottom-center
// of the screen.
type DarwinOverlay struct {
	initialized bool
}

// NewDarwinOverlay creates and initializes the macOS overlay window.
func NewDarwinOverlay() *DarwinOverlay {
	C.overlayInit()
	return &DarwinOverlay{initialized: true}
}

func (o *DarwinOverlay) Show() {
	if o.initialized {
		C.overlayShow()
	}
}

func (o *DarwinOverlay) Hide() {
	if o.initialized {
		C.overlayHide()
	}
}

func (o *DarwinOverlay) UpdateText(text string) {
	if !o.initialized {
		return
	}
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	C.overlayUpdateText(cs)
}

func (o *DarwinOverlay) SetStatus(status Status) {
	if !o.initialized {
		return
	}
	cs := C.CString(status.String())
	defer C.free(unsafe.Pointer(cs))
	C.overlaySetStatus(cs)
}
