//go:build darwin

package notify

/*
#cgo LDFLAGS: -framework Foundation
#include <stdlib.h>

void notifySend(const char *title, const char *body);
*/
import "C"

import "unsafe"

// DarwinNotifier sends macOS notifications via NSUserNotification.
type DarwinNotifier struct{}

// NewDarwinNotifier creates a macOS notifier.
func NewDarwinNotifier() *DarwinNotifier { return &DarwinNotifier{} }

func (n *DarwinNotifier) Send(title, body string) {
	ct := C.CString(title)
	cb := C.CString(body)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	C.notifySend(ct, cb)
}
