// Motion-triggered location sharing on iOS, the twin of Android's
// MotionTracker.kt, behind the same "Motion" module (src/location/motion.ts).
//
// iOS never lets CoreMotion wake an app, so the wake-ups come from Core
// Location, which can relaunch the app even after iOS has closed it:
//  - a region (geofence, CLCircularRegion) around where you stopped: leaving
//    it starts tracking, as on Android;
//  - significant-location-change monitoring (cell towers, about 500 m), the
//    backstop for a missed region exit and the only wake-up when Precise
//    Location is off (iOS doesn't monitor regions then).
// Tracking is standard location updates with allowsBackgroundLocationUpdates
// (UIBackgroundModes: location), sending a fix at most every 5 s to
// /locations/batch, until you've been still for a while: the same 75 m /
// 5 min rule as Android, or 2 min once Motion & Fitness (CMMotionActivity)
// says you're stationary too. Then it drops a new region where you are and
// stops updates, and iOS suspends the app.
//
// While tracking, iOS shows its blue location indicator
// (showsBackgroundLocationIndicator), the counterpart of Android's
// "Sharing your location while you're moving" notification: background
// location use is always visible to you, as App Review expects.
//
// Needs "Always" location. With "While Using" only, nothing here runs and
// friends see you only while the app is open (BackgroundLocationBanner.tsx
// says so). A force-quit app (swiped away in the app switcher) isn't
// relaunched by iOS for any of this until you open it again.
//
// Turned on from motion.ts with the session token (kept in the Keychain,
// readable after the first unlock so a relaunch in the background can use
// it); the server URL is baked in (SuperAtlasApiUrl, Info.plist). Signing out
// turns it off, and so does a 401 from the server.

#import <CoreLocation/CoreLocation.h>
#import <CoreMotion/CoreMotion.h>
#import <React/RCTBridgeModule.h>
#import <Security/Security.h>
#import <UIKit/UIKit.h>

static NSString *const kFenceId = @"superatlas-still";
static NSString *const kKeychainService = @"com.superatlasapp.motion";
static NSString *const kKeychainAccount = @"token";
// Apple's advice is 100 m at least; smaller regions miss exits.
static const CLLocationDistance kFenceRadiusM = 150;
// About as often as Life360 sends on the move.
static const NSTimeInterval kSendIntervalS = 5;
static const CLLocationDistance kMinDistanceM = 10;
// Worse than this is a cell-tower guess, not a position (ignored with
// Precise Location off, where every fix is that coarse).
static const CLLocationAccuracy kMaxAccuracyM = 100;
// Still means staying within this of one spot for this long: longer than a
// red light, shorter than parking and walking off.
static const CLLocationDistance kStillRadiusM = 75;
static const NSTimeInterval kStillS = 5 * 60;
// ...or this long once CoreMotion agrees you're stationary.
static const NSTimeInterval kMotionStillS = 2 * 60;
static const NSTimeInterval kCheckS = 30;

#pragma mark - Keychain

static NSDictionary *TokenQuery(void)
{
  return @{
    (__bridge id)kSecClass : (__bridge id)kSecClassGenericPassword,
    (__bridge id)kSecAttrService : kKeychainService,
    (__bridge id)kSecAttrAccount : kKeychainAccount,
  };
}

static NSString *ReadToken(void)
{
  NSMutableDictionary *q = [TokenQuery() mutableCopy];
  q[(__bridge id)kSecReturnData] = @YES;
  q[(__bridge id)kSecMatchLimit] = (__bridge id)kSecMatchLimitOne;
  CFTypeRef data = NULL;
  if (SecItemCopyMatching((__bridge CFDictionaryRef)q, &data) != errSecSuccess || !data) return nil;
  return [[NSString alloc] initWithData:(__bridge_transfer NSData *)data encoding:NSUTF8StringEncoding];
}

static void WriteToken(NSString *token)
{
  SecItemDelete((__bridge CFDictionaryRef)TokenQuery());
  if (!token) return;
  NSMutableDictionary *q = [TokenQuery() mutableCopy];
  q[(__bridge id)kSecValueData] = [token dataUsingEncoding:NSUTF8StringEncoding];
  q[(__bridge id)kSecAttrAccessible] = (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly;
  SecItemAdd((__bridge CFDictionaryRef)q, NULL);
}

#pragma mark - Tracker

@interface SAMotionTracker : NSObject <CLLocationManagerDelegate>
+ (instancetype)shared;
- (void)enableWithToken:(NSString *)token;
- (void)disable;
- (void)requestActivity:(void (^)(BOOL granted))done;
@end

@implementation SAMotionTracker {
  CLLocationManager *_manager;
  CMMotionActivityManager *_activity;
  NSOperationQueue *_activityQueue;
  NSString *_token;
  BOOL _tracking;
  // A one-off fix was asked for, to put the region around.
  BOOL _wantFence;
  // Where you've been within kStillRadiusM of since _stillSince.
  CLLocation *_anchor;
  NSDate *_stillSince;
  // Since when CoreMotion has said stationary (nil: moving, or unknown).
  NSDate *_motionStillSince;
  CLLocation *_latest;
  NSDate *_lastSent;
  NSTimer *_checkTimer;
}

+ (instancetype)shared
{
  static SAMotionTracker *shared;
  static dispatch_once_t once;
  dispatch_once(&once, ^{
    shared = [SAMotionTracker new];
  });
  return shared;
}

// Created as launch finishes, including when iOS relaunches the app in the
// background for a region exit or a significant change: those events are
// delivered once a location manager with a delegate exists.
+ (void)load
{
  [[NSNotificationCenter defaultCenter] addObserverForName:UIApplicationDidFinishLaunchingNotification
                                                    object:nil
                                                     queue:[NSOperationQueue mainQueue]
                                                usingBlock:^(NSNotification *note) {
    [[SAMotionTracker shared] arm];
  }];
}

- (instancetype)init
{
  if ((self = [super init])) {
    _manager = [CLLocationManager new];
    _manager.delegate = self;
    _manager.desiredAccuracy = kCLLocationAccuracyBest;
    _manager.distanceFilter = kMinDistanceM;
    _manager.activityType = CLActivityTypeOther;
    // Only this decides when to stop: iOS's own pausing never resumes by
    // itself in the background.
    _manager.pausesLocationUpdatesAutomatically = NO;
    _manager.showsBackgroundLocationIndicator = YES;
    _activity = [CMMotionActivityManager new];
    _activityQueue = [NSOperationQueue new];
    _activityQueue.maxConcurrentOperationCount = 1;
    _token = ReadToken();
  }
  return self;
}

- (BOOL)enabled
{
  return _token != nil;
}

- (BOOL)hasAlways
{
  return _manager.authorizationStatus == kCLAuthorizationStatusAuthorizedAlways;
}

- (BOOL)precise
{
  return _manager.accuracyAuthorization == CLAccuracyAuthorizationFullAccuracy;
}

- (BOOL)hasMotion
{
  return [CMMotionActivityManager isActivityAvailable] &&
         [CMMotionActivityManager authorizationStatus] == CMAuthorizationStatusAuthorized;
}

- (void)enableWithToken:(NSString *)token
{
  _token = token;
  WriteToken(token);
  [self arm];
}

- (void)disable
{
  _token = nil;
  WriteToken(nil);
  [self stopTracking];
  [_manager stopMonitoringSignificantLocationChanges];
  for (CLRegion *region in _manager.monitoredRegions) {
    if ([region.identifier isEqualToString:kFenceId]) [_manager stopMonitoringForRegion:region];
  }
}

/** Starts the wake-ups, and puts the region where you are unless you're
 *  being tracked as moving. Safe to call any time. */
- (void)arm
{
  if (!self.enabled || !self.hasAlways) return;
  [_manager startMonitoringSignificantLocationChanges];
  if (_tracking) return;
  _wantFence = YES;
  [_manager requestLocation];
}

/** Puts the region around here (same identifier, so it replaces the last). */
- (void)fenceAt:(CLLocation *)here
{
  if (!self.hasAlways || !self.precise ||
      ![CLLocationManager isMonitoringAvailableForClass:[CLCircularRegion class]]) {
    return;
  }
  // A fix that's only good to 300 m needs a region at least that big, or
  // GPS noise alone would count as leaving.
  CLLocationDistance radius = MIN(MAX(kFenceRadiusM, here.horizontalAccuracy), _manager.maximumRegionMonitoringDistance);
  CLCircularRegion *fence = [[CLCircularRegion alloc] initWithCenter:here.coordinate radius:radius identifier:kFenceId];
  fence.notifyOnEntry = NO;
  fence.notifyOnExit = YES;
  [_manager startMonitoringForRegion:fence];
}

/** Whether here is outside the region, or there isn't one. */
- (BOOL)outsideFence:(CLLocation *)here
{
  for (CLRegion *region in _manager.monitoredRegions) {
    if (![region.identifier isEqualToString:kFenceId] || ![region isKindOfClass:[CLCircularRegion class]]) continue;
    CLCircularRegion *fence = (CLCircularRegion *)region;
    CLLocation *center = [[CLLocation alloc] initWithLatitude:fence.center.latitude longitude:fence.center.longitude];
    return [here distanceFromLocation:center] > fence.radius;
  }
  return YES;
}

- (void)startTracking
{
  if (_tracking || !self.enabled || !self.hasAlways) return;
  _tracking = YES;
  _wantFence = NO;
  _anchor = nil;
  _stillSince = [NSDate date];
  _motionStillSince = nil;
  _manager.allowsBackgroundLocationUpdates = YES;
  [_manager startUpdatingLocation];
  if (self.hasMotion) {
    __weak SAMotionTracker *weakSelf = self;
    [_activity startActivityUpdatesToQueue:_activityQueue withHandler:^(CMMotionActivity *activity) {
      dispatch_async(dispatch_get_main_queue(), ^{
        [weakSelf onActivity:activity];
      });
    }];
  }
  [_checkTimer invalidate];
  _checkTimer = [NSTimer scheduledTimerWithTimeInterval:kCheckS
                                                 target:self
                                               selector:@selector(checkStill)
                                               userInfo:nil
                                                repeats:YES];
}

- (void)stopTracking
{
  if (!_tracking) return;
  _tracking = NO;
  [_checkTimer invalidate];
  _checkTimer = nil;
  [_activity stopActivityUpdates];
  [_manager stopUpdatingLocation];
  _manager.allowsBackgroundLocationUpdates = NO;
}

- (void)onActivity:(CMMotionActivity *)activity
{
  if (!activity) return;
  BOOL still = activity.stationary && !activity.automotive && activity.confidence != CMMotionActivityConfidenceLow;
  if (!still) {
    _motionStillSince = nil;
  } else if (!_motionStillSince) {
    _motionStillSince = [NSDate date];
  }
}

// On a timer rather than per fix, since a phone that isn't moving gets no
// fixes at all (distanceFilter).
- (void)checkStill
{
  NSTimeInterval still = -[_stillSince timeIntervalSinceNow];
  BOOL motionSaysStill = _motionStillSince && -[_motionStillSince timeIntervalSinceNow] >= kMotionStillS;
  if (still < kStillS && !(motionSaysStill && still >= kMotionStillS)) return;
  CLLocation *here = _latest;
  [self stopTracking];
  if (here) {
    [self fenceAt:here];
  } else {
    [self arm];
  }
}

#pragma mark CLLocationManagerDelegate

- (void)locationManager:(CLLocationManager *)manager didUpdateLocations:(NSArray<CLLocation *> *)locations
{
  CLLocation *fix = locations.lastObject;
  if (!fix || fix.horizontalAccuracy < 0) return;

  if (_tracking) {
    if (self.precise && fix.horizontalAccuracy > kMaxAccuracyM) return;
    _latest = fix;
    if (!_anchor || [fix distanceFromLocation:_anchor] > kStillRadiusM) {
      _anchor = fix;
      _stillSince = [NSDate date];
    }
    if (!_lastSent || -[_lastSent timeIntervalSinceNow] >= kSendIntervalS) [self upload:fix];
    return;
  }

  if (_wantFence) {
    // The one-off fix from arm.
    _wantFence = NO;
    [self fenceAt:fix];
    return;
  }

  // A significant change while still: a trip, if it's taken you out of the
  // region (or there's no region to be out of).
  if (self.enabled && [self outsideFence:fix]) {
    [self upload:fix];
    [self startTracking];
  }
}

- (void)locationManager:(CLLocationManager *)manager didExitRegion:(CLRegion *)region
{
  if ([region.identifier isEqualToString:kFenceId] && self.enabled) [self startTracking];
}

- (void)locationManager:(CLLocationManager *)manager didFailWithError:(NSError *)error
{
  // A failed one-off fix: the significant-change monitoring still wakes us.
  _wantFence = NO;
}

- (void)locationManager:(CLLocationManager *)manager
    monitoringDidFailForRegion:(CLRegion *)region
                     withError:(NSError *)error
{
}

- (void)locationManagerDidChangeAuthorization:(CLLocationManager *)manager
{
  if (self.hasAlways) {
    [self arm];
  } else {
    // "Always" taken away (or downgraded to While Using) in Settings.
    [self stopTracking];
  }
}

#pragma mark Upload

/** Sends one fix to /locations/batch. A 401 means signed out elsewhere:
 *  stop for good until the next sign-in. */
- (void)upload:(CLLocation *)fix
{
  NSString *token = _token;
  NSString *base = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"SuperAtlasApiUrl"];
  if (!token || base.length == 0) return;
  _lastSent = [NSDate date];

  static NSISO8601DateFormatter *iso;
  static dispatch_once_t once;
  dispatch_once(&once, ^{
    iso = [NSISO8601DateFormatter new];
  });
  NSMutableDictionary *point = [@{
    @"latitude" : @(fix.coordinate.latitude),
    @"longitude" : @(fix.coordinate.longitude),
    @"accuracyM" : @(fix.horizontalAccuracy),
    @"recordedAt" : [iso stringFromDate:fix.timestamp],
  } mutableCopy];
  if (fix.speed >= 0) point[@"speedMps"] = @(fix.speed);
  NSData *body = [NSJSONSerialization dataWithJSONObject:@{@"points" : @[ point ]} options:0 error:nil];

  NSMutableURLRequest *req = [NSMutableURLRequest
      requestWithURL:[NSURL URLWithString:[base stringByAppendingString:@"/locations/batch"]]];
  req.HTTPMethod = @"POST";
  req.timeoutInterval = 15;
  req.HTTPBody = body;
  [req setValue:@"application/json" forHTTPHeaderField:@"Content-Type"];
  [req setValue:[@"Bearer " stringByAppendingString:token] forHTTPHeaderField:@"Authorization"];

  // Woken in the background, the app gets a few seconds unless tracking
  // keeps it running; ask for enough to finish this.
  UIApplication *app = [UIApplication sharedApplication];
  __block UIBackgroundTaskIdentifier task = [app beginBackgroundTaskWithName:@"superatlas-upload"
                                                           expirationHandler:^{
    [app endBackgroundTask:task];
    task = UIBackgroundTaskInvalid;
  }];
  [[[NSURLSession sharedSession] dataTaskWithRequest:req
                                   completionHandler:^(NSData *data, NSURLResponse *response, NSError *error) {
    NSInteger status = [response isKindOfClass:[NSHTTPURLResponse class]] ? ((NSHTTPURLResponse *)response).statusCode : -1;
    dispatch_async(dispatch_get_main_queue(), ^{
      if (status == 401 && [token isEqualToString:self->_token]) [self disable];
      if (task != UIBackgroundTaskInvalid) {
        [app endBackgroundTask:task];
        task = UIBackgroundTaskInvalid;
      }
    });
  }] resume];
}

#pragma mark Motion & Fitness

/** iOS has no call that just asks: the first activity query shows the
 *  prompt. */
- (void)requestActivity:(void (^)(BOOL granted))done
{
  if (![CMMotionActivityManager isActivityAvailable]) {
    done(NO);
    return;
  }
  if ([CMMotionActivityManager authorizationStatus] != CMAuthorizationStatusNotDetermined) {
    done(self.hasMotion);
    return;
  }
  NSDate *now = [NSDate date];
  [_activity queryActivityStartingFromDate:[now dateByAddingTimeInterval:-60]
                                    toDate:now
                                   toQueue:[NSOperationQueue mainQueue]
                               withHandler:^(NSArray<CMMotionActivity *> *activities, NSError *error) {
    done(self.hasMotion);
  }];
}

@end

#pragma mark - JS module

@interface MotionModule : NSObject <RCTBridgeModule>
@end

@implementation MotionModule

RCT_EXPORT_MODULE(Motion)

+ (BOOL)requiresMainQueueSetup
{
  return NO;
}

- (dispatch_queue_t)methodQueue
{
  return dispatch_get_main_queue();
}

RCT_EXPORT_METHOD(enable:(NSString *)token resolve:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  [[SAMotionTracker shared] enableWithToken:token];
  resolve(nil);
}

RCT_EXPORT_METHOD(disable:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  [[SAMotionTracker shared] disable];
  resolve(nil);
}

/** Asks for Motion & Fitness (motion.ts requestActivityPermission). */
RCT_EXPORT_METHOD(requestActivity:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  [[SAMotionTracker shared] requestActivity:^(BOOL granted) {
    resolve(@(granted));
  }];
}

@end
