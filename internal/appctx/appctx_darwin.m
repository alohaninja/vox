#import <AppKit/AppKit.h>
#include <stdlib.h>
#include <string.h>

// getFrontmostAppBundleID returns the bundle identifier of the frontmost app.
// The caller must free() the returned string.
const char* getFrontmostAppBundleID(void) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        NSString *bid = [app bundleIdentifier];
        if (bid == nil) return NULL;
        return strdup([bid UTF8String]);
    }
}

// getFrontmostAppName returns the localized name of the frontmost app.
// The caller must free() the returned string.
const char* getFrontmostAppName(void) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        NSString *name = [app localizedName];
        if (name == nil) return NULL;
        return strdup([name UTF8String]);
    }
}
