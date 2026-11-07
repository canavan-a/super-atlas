// Tells JS which build this is (src/distribution.ts), the twin of Android's
// DistributionModule.kt: "appstore" (App Store / TestFlight: FCM friend
// notifications, no in-app updater), and which server it was built for
// (SUPERATLAS_API_URL, a build setting, through Info.plist).

#import <React/RCTBridgeModule.h>

@interface DistributionModule : NSObject <RCTBridgeModule>
@end

@implementation DistributionModule

RCT_EXPORT_MODULE(Distribution)

+ (BOOL)requiresMainQueueSetup
{
  return NO;
}

- (NSDictionary *)constantsToExport
{
  NSString *apiUrl = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"SuperAtlasApiUrl"] ?: @"";
  return @{@"flavor" : @"appstore", @"apiUrl" : apiUrl};
}

@end
