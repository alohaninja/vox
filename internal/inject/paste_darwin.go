//go:build darwin

package inject

/*
#cgo LDFLAGS: -framework ApplicationServices -framework AppKit -framework Carbon
#include <ApplicationServices/ApplicationServices.h>
#include <Carbon/Carbon.h>

void *clipboardSnapshotCreate(void);
int clipboardSnapshotRestore(void *snapshot);
void clipboardSnapshotFree(void *snapshot);

// Returned in place of a keycode when the active keyboard layout cannot be
// read, so callers post nothing rather than guessing at a physical key.
#define pasteKeyUnknown (-1)

// charInLayout returns the character a keycode produces in the given layout
// while Command is held, or 0 if it produces no single character.
static UniChar charInLayout(const UCKeyboardLayout *layout, CGKeyCode keycode) {
    UInt32 deadKeyState = 0;
    UniChar chars[4];
    UniCharCount len = 0;
    OSStatus status = UCKeyTranslate(
        layout, keycode, kUCKeyActionDown, (cmdKey >> 8) & 0xFF, LMGetKbdType(),
        kUCKeyTranslateNoDeadKeysMask, &deadKeyState, 4, &len, chars
    );
    if (status != noErr || len != 1) {
        return 0;
    }
    return chars[0];
}

// activeKeyboardLayout returns the active ASCII-capable keyboard layout, or
// NULL if it cannot be read. On success the caller owns *src and must release
// it; the returned layout points into *src and dies with it. Callers resolve
// the layout once so a mid-call input source switch cannot split their answer
// across two layouts.
static const UCKeyboardLayout *activeKeyboardLayout(TISInputSourceRef *src) {
    *src = TISCopyCurrentASCIICapableKeyboardLayoutInputSource();
    if (*src == NULL) {
        return NULL;
    }

    CFDataRef data = (CFDataRef)TISGetInputSourceProperty(*src, kTISPropertyUnicodeKeyLayoutData);
    if (data == NULL) {
        CFRelease(*src);
        *src = NULL;
        return NULL;
    }

    return (const UCKeyboardLayout *)CFDataGetBytePtr(data);
}

// pasteKeycodeInLayout returns the key that produces 'v' in the given layout,
// or pasteKeyUnknown if no key does.
static int pasteKeycodeInLayout(const UCKeyboardLayout *layout) {
    for (CGKeyCode keycode = 0; keycode < 128; keycode++) {
        if (charInLayout(layout, keycode) == 'v') {
            return keycode;
        }
    }
    return pasteKeyUnknown;
}

// pasteKeycode returns the key that produces 'v' under the active keyboard
// layout, or pasteKeyUnknown if that layout cannot be read. Applications match
// Cmd+V by character rather than by keycode, and layouts such as Dvorak put 'v'
// on a different physical key than QWERTY.
// The lookup runs per paste so switching input sources takes effect at once.
int pasteKeycode(void) {
    TISInputSourceRef src = NULL;
    const UCKeyboardLayout *layout = activeKeyboardLayout(&src);
    if (layout == NULL) {
        return pasteKeyUnknown;
    }

    int keycode = pasteKeycodeInLayout(layout);
    CFRelease(src);
    return keycode;
}

// charForPasteKeycode returns the character the resolved paste key produces
// while Command is held, or 0 if the active layout cannot be read.
UniChar charForPasteKeycode(void) {
    TISInputSourceRef src = NULL;
    const UCKeyboardLayout *layout = activeKeyboardLayout(&src);
    if (layout == NULL) {
        return 0;
    }

    int keycode = pasteKeycodeInLayout(layout);
    UniChar ch = (keycode == pasteKeyUnknown) ? 0 : charInLayout(layout, (CGKeyCode)keycode);
    CFRelease(src);
    return ch;
}

// simulateCmdV posts Cmd+V keyboard events to the system. It returns 0 without
// posting anything when the active keyboard layout cannot be read, since any
// keycode picked blind would fire whatever unrelated shortcut sits there, or
// when the events cannot be allocated.
int simulateCmdV(void) {
    int keycode = pasteKeycode();
    if (keycode == pasteKeyUnknown) {
        return 0;
    }

    CGKeyCode v = (CGKeyCode)keycode;
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, v, true);
    CGEventRef keyUp   = CGEventCreateKeyboardEvent(NULL, v, false);
    if (keyDown == NULL || keyUp == NULL) {
        if (keyDown != NULL) {
            CFRelease(keyDown);
        }
        if (keyUp != NULL) {
            CFRelease(keyUp);
        }
        return 0;
    }

    CGEventSetFlags(keyDown, kCGEventFlagMaskCommand);
    CGEventSetFlags(keyUp, kCGEventFlagMaskCommand);

    CGEventPost(kCGHIDEventTap, keyDown);
    CGEventPost(kCGHIDEventTap, keyUp);

    CFRelease(keyDown);
    CFRelease(keyUp);
    return 1;
}
*/
import "C"

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TypeText copies the given text to the system clipboard and simulates Cmd+V
// in the currently focused macOS application. Empty text is a no-op.
func TypeText(text string) error {
	if text == "" {
		return nil
	}

	snapshot := C.clipboardSnapshotCreate()
	if snapshot == nil {
		return fmt.Errorf("inject: snapshot clipboard failed")
	}
	defer C.clipboardSnapshotFree(snapshot)

	if err := copyToClipboard(text); err != nil {
		return fmt.Errorf("inject: copy to clipboard: %w", err)
	}

	// Brief delay to ensure clipboard contents are committed.
	time.Sleep(50 * time.Millisecond)

	// Post Cmd+V via CGEvent (works regardless of which app is focused).
	// When the paste key is unknown the text stays on the clipboard for the user
	// to paste by hand, as in auto-paste off mode, so it is not lost.
	if C.simulateCmdV() != 1 {
		return ErrPasteKeyUnknown
	}

	// Give the focused app time to consume the paste, then restore user clipboard.
	time.Sleep(250 * time.Millisecond)
	if C.clipboardSnapshotRestore(snapshot) != 1 {
		return fmt.Errorf("inject: restore clipboard failed")
	}

	return nil
}

// pasteKeycode reports the virtual keycode that TypeText posts Cmd+V to under
// the active keyboard layout. A negative result means the layout could not be
// read and no paste was posted.
func pasteKeycode() int {
	return int(C.pasteKeycode())
}

// charForPasteKeycode reports the character the paste key produces with Command
// held. Zero means the active keyboard layout could not be read.
func charForPasteKeycode() rune {
	return rune(C.charForPasteKeycode())
}

// ReadClipboard returns the current contents of the system clipboard.
// Trailing newlines are trimmed for consistency with clipboard semantics.
func ReadClipboard() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// ClearClipboard writes an empty string to the system clipboard.
func ClearClipboard() error {
	return copyToClipboard("")
}

// CopyToClipboard writes text to the system clipboard without simulating a
// paste keystroke. Used by the "auto-paste off" mode: the user gets the
// transcription on their clipboard and pastes wherever they want.
func CopyToClipboard(text string) error {
	if text == "" {
		return nil
	}
	return copyToClipboard(text)
}

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pbcopy: %w (output: %s)", err, string(output))
	}
	return nil
}
