#import <Foundation/Foundation.h>
#include <stdlib.h>

// notifySend posts a native macOS notification using NSUserNotification.
// This API is deprecated in macOS 11+ but still works and avoids the
// entitlement requirements of UNUserNotificationCenter. For a developer
// tool that doesn't ship through the App Store, this is the pragmatic choice.
void notifySend(const char *title, const char *body) {
    if (title == NULL || body == NULL) return;

    NSString *nsTitle = [NSString stringWithUTF8String:title];
    NSString *nsBody  = [NSString stringWithUTF8String:body];

    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
            NSUserNotification *notification = [[NSUserNotification alloc] init];
            notification.title = nsTitle;
            notification.informativeText = nsBody;
            notification.soundName = nil; // silent — vox has its own sounds

            [[NSUserNotificationCenter defaultUserNotificationCenter]
                deliverNotification:notification];
#pragma clang diagnostic pop
        }
    });
}
