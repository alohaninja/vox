#include <ApplicationServices/ApplicationServices.h>
#include "_cgo_export.h"

static CFMachPortRef _tap = NULL;

static CGEventRef hotkey_callback(
    CGEventTapProxy proxy,
    CGEventType type,
    CGEventRef event,
    void *refcon
) {
    if (type == kCGEventTapDisabledByTimeout) {
        if (_tap) CGEventTapEnable(_tap, true);
        return event;
    }
    if (type == kCGEventTapDisabledByUserInput) {
        // Accessibility permission was revoked mid-session. Re-enabling
        // won't help (the system won't allow it), but we log through Go
        // so the user can see a diagnostic in the menubar log.
        return event;
    }

    CGEventFlags flags = CGEventGetFlags(event);
    int64_t keycode = CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);

    GoInt32 consume = goEventCallback(
        (GoUint32)type,
        (GoUint64)flags,
        (GoInt64)keycode
    );

    if (consume == 1) {
        return NULL;
    }
    return event;
}

int checkAccessibility(int prompt) {
    if (!prompt) {
        return AXIsProcessTrusted() ? 1 : 0;
    }

    const void *keys[] = { kAXTrustedCheckOptionPrompt };
    const void *values[] = { kCFBooleanTrue };
    CFDictionaryRef options = CFDictionaryCreate(
        kCFAllocatorDefault,
        keys,
        values,
        1,
        &kCFCopyStringDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks
    );
    if (!options) {
        return AXIsProcessTrusted() ? 1 : 0;
    }

    Boolean trusted = AXIsProcessTrustedWithOptions(options);
    CFRelease(options);
    return trusted ? 1 : 0;
}

// startEventTap creates the CGEventTap and registers its run-loop source on
// the *main* run loop, then returns immediately. The caller is responsible
// for running the main run loop (typically via [NSApp run] in the UI layer).
int startEventTap(void) {
    CGEventMask mask = CGEventMaskBit(kCGEventFlagsChanged) |
                       CGEventMaskBit(kCGEventKeyDown) |
                       CGEventMaskBit(kCGEventKeyUp);

    _tap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        mask,
        hotkey_callback,
        NULL
    );

    if (!_tap) return -1;

    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(
        kCFAllocatorDefault, _tap, 0
    );
    // Use the main run loop so the source is alive whichever thread Start()
    // happens to be invoked from. NSApp's main loop will dispatch events.
    CFRunLoopAddSource(CFRunLoopGetMain(), source, kCFRunLoopCommonModes);
    // CFRunLoopAddSource retains; balance the Create rule's implicit retain.
    CFRelease(source);
    CGEventTapEnable(_tap, true);

    return 0;
}
