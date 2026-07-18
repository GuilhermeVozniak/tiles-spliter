#import <Cocoa/Cocoa.h>

typedef struct { unsigned int id; double fx, fy, fw, fh, vx, vy, vw, vh; } TSDisplay;

// Fills out[] with all screens in top-left-origin coordinates. Returns count.
static int ts_displays(TSDisplay *out, int max) {
  NSArray<NSScreen *> *screens = [NSScreen screens];
  if (screens.count == 0) return 0;
  double primaryH = NSHeight(screens[0].frame);
  int n = 0;
  for (NSScreen *s in screens) {
    if (n >= max) break;
    NSRect f = s.frame, v = s.visibleFrame;
    TSDisplay d;
    d.id = [s.deviceDescription[@"NSScreenNumber"] unsignedIntValue];
    d.fx = f.origin.x; d.fy = primaryH - (f.origin.y + f.size.height); d.fw = f.size.width; d.fh = f.size.height;
    d.vx = v.origin.x; d.vy = primaryH - (v.origin.y + v.size.height); d.vw = v.size.width; d.vh = v.size.height;
    out[n++] = d;
  }
  return n;
}
