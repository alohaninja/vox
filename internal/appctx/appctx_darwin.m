#import <AppKit/AppKit.h>
#include <stdlib.h>
#include <string.h>

// FrontmostApp holds both the bundle ID and name from a single frontmostApplication snapshot.
typedef struct {
    const char* bundleID;
    const char* name;
} FrontmostApp;

// safeStrdup copies the UTF8 representation of an NSString, returning NULL
// if the string is nil or UTF8String fails.
static const char* safeStrdup(NSString *s) {
    if (s == nil) return NULL;
    const char *utf = [s UTF8String];
    if (utf == NULL) return NULL;
    return strdup(utf);
}

// getFrontmostApp atomically captures both the bundle identifier and localized
// name from the same NSRunningApplication instance.
// The caller must free() each non-NULL field.
FrontmostApp getFrontmostApp(void) {
    FrontmostApp result = {NULL, NULL};
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (app == nil) return result;
        result.bundleID = safeStrdup([app bundleIdentifier]);
        result.name = safeStrdup([app localizedName]);
    }
    return result;
}
