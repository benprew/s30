#import "AppDelegate.h"
#import "MobileEbitenViewControllerWithErrorHandling.h"
#import <Mobile/Mobile.h>

@implementation AppDelegate

- (BOOL)application:(UIApplication *)application didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
    NSArray *paths = NSSearchPathForDirectoriesInDomains(NSDocumentDirectory, NSUserDomainMask, YES);
    NSString *documentsDirectory = [paths firstObject];
    MobileSetSaveDir(documentsDirectory);
    return YES;
}

- (void)applicationWillResignActive:(UIApplication *)application {
    [[self ebitenViewController] suspendGame];
    MobileSaveGame();
}

- (void)applicationDidEnterBackground:(UIApplication *)application {
}

- (void)applicationWillEnterForeground:(UIApplication *)application {
}

- (void)applicationDidBecomeActive:(UIApplication *)application {
    [[self ebitenViewController] resumeGame];
}

- (void)applicationWillTerminate:(UIApplication *)application {
    MobileSaveGame();
}

- (MobileEbitenViewControllerWithErrorHandling*)ebitenViewController {
    return (MobileEbitenViewControllerWithErrorHandling*)(self.window.rootViewController);
}

@end
