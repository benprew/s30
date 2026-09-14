#import "MobileEbitenViewControllerWithErrorHandling.h"
#import <Foundation/Foundation.h>

@implementation MobileEbitenViewControllerWithErrorHandling

- (void)onErrorOnGameUpdate:(NSError*)err {
    NSLog(@"S30 error during game update: %@", err);
}

@end
