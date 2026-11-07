// Friend notifications on iOS (src/notifications.ts), the twin of the Google
// Play build's PushModule.kt: an FCM registration token, registered with the
// server (PUT /push/token), which sends to it through FCM and APNs
// (server/internal/push). The notifications are plain alerts, so iOS shows
// them by itself while the app is in the background; while it's open, the
// delegate below shows them as banners.
//
// Needs SuperAtlasApp/GoogleService-Info.plist (Firebase console: add an
// iOS app with bundle ID com.superatlasapp; upload an APNs key there too).
// Without it the build still works, with no notifications: isAvailable says
// no. The "Copy GoogleService-Info.plist" build phase copies it in if present.

#import <FirebaseCore/FirebaseCore.h>
#import <FirebaseMessaging/FirebaseMessaging.h>
#import <React/RCTBridgeModule.h>
#import <UIKit/UIKit.h>
#import <UserNotifications/UserNotifications.h>

static BOOL FirebaseConfigured(void)
{
  return [FIRApp defaultApp] != nil;
}

@interface PushCenterDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation PushCenterDelegate

+ (instancetype)shared
{
  static PushCenterDelegate *shared;
  static dispatch_once_t once;
  dispatch_once(&once, ^{
    shared = [PushCenterDelegate new];
  });
  return shared;
}

+ (void)load
{
  [[NSNotificationCenter defaultCenter] addObserverForName:UIApplicationDidFinishLaunchingNotification
                                                    object:nil
                                                     queue:nil
                                                usingBlock:^(NSNotification *note) {
    if (FirebaseConfigured()) return;
    if ([[NSBundle mainBundle] pathForResource:@"GoogleService-Info" ofType:@"plist"]) [FIRApp configure];
  }];
  [UNUserNotificationCenter currentNotificationCenter].delegate = [PushCenterDelegate shared];
}

// App open: show it anyway, as Android does.
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler
{
  completionHandler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionSound |
                    UNNotificationPresentationOptionList);
}

// Tapped: opening the app is all it takes, the friends list refreshes itself.
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
    didReceiveNotificationResponse:(UNNotificationResponse *)response
             withCompletionHandler:(void (^)(void))completionHandler
{
  completionHandler();
}

@end

@interface PushModule : NSObject <RCTBridgeModule>
@end

@implementation PushModule

RCT_EXPORT_MODULE(Push)

+ (BOOL)requiresMainQueueSetup
{
  return NO;
}

/** Whether notifications can work: Firebase is set up in this build, and
 *  you allow them (asked once; iOS remembers the answer). */
RCT_EXPORT_METHOD(isAvailable:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  if (!FirebaseConfigured()) {
    resolve(@NO);
    return;
  }
  UNAuthorizationOptions options = UNAuthorizationOptionAlert | UNAuthorizationOptionSound | UNAuthorizationOptionBadge;
  [[UNUserNotificationCenter currentNotificationCenter]
      requestAuthorizationWithOptions:options
                    completionHandler:^(BOOL granted, NSError *error) {
    if (granted) {
      dispatch_async(dispatch_get_main_queue(), ^{
        [[UIApplication sharedApplication] registerForRemoteNotifications];
      });
    }
    resolve(@(granted));
  }];
}

/** The FCM token, once APNs has given the device its token (Firebase picks
 *  that up itself). Waits up to 20 s for it. */
RCT_EXPORT_METHOD(getToken:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  if (!FirebaseConfigured()) {
    reject(@"push", @"Firebase isn't set up in this build", nil);
    return;
  }
  [self waitForAPNs:40 then:^{
    [[FIRMessaging messaging] tokenWithCompletion:^(NSString *token, NSError *error) {
      if (token) resolve(token);
      else reject(@"push", error.localizedDescription ?: @"no FCM token", error);
    }];
  }];
}

- (void)waitForAPNs:(int)tries then:(void (^)(void))next
{
  if ([FIRMessaging messaging].APNSToken || tries <= 0) {
    next();
    return;
  }
  dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.5 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
    [self waitForAPNs:tries - 1 then:next];
  });
}

@end
