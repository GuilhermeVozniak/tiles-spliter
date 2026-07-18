#import <Cocoa/Cocoa.h>

static NSPanel *ts_overlay = nil;

// x,y,w,h in top-left-origin global coords; converted to Cocoa here.
static void ts_overlay_show(double x, double y, double w, double h) {
  dispatch_async(dispatch_get_main_queue(), ^{
    double primaryH = NSHeight([NSScreen screens][0].frame);
    NSRect frame = NSMakeRect(x, primaryH - y - h, w, h);
    if (!ts_overlay) {
      ts_overlay = [[NSPanel alloc] initWithContentRect:frame
                                              styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                backing:NSBackingStoreBuffered
                                                  defer:NO];
      ts_overlay.level = NSStatusWindowLevel;
      ts_overlay.opaque = NO;
      ts_overlay.ignoresMouseEvents = YES;
      ts_overlay.hasShadow = NO;
      ts_overlay.backgroundColor = [NSColor clearColor];
      ts_overlay.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorTransient;
      NSView *v = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, w, h)];
      v.wantsLayer = YES;
      v.layer.backgroundColor = [[NSColor colorWithWhite:0.85 alpha:0.25] CGColor];
      v.layer.borderColor = [[NSColor colorWithWhite:0.9 alpha:0.6] CGColor];
      v.layer.borderWidth = 2;
      v.layer.cornerRadius = 8;
      v.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
      ts_overlay.contentView = v;
    }
    [ts_overlay setFrame:frame display:YES];
    [ts_overlay orderFrontRegardless];
  });
}

static void ts_overlay_hide(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [ts_overlay orderOut:nil];
  });
}
