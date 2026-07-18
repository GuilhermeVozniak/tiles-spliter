#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>

extern void goDragEvent(int kind, double x, double y, int modThirds); // exported from Go

static CFMachPortRef ts_tap = NULL;
static CFRunLoopSourceRef ts_tap_source = NULL;

static CGEventRef ts_tap_cb(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *info) {
  if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
    if (ts_tap) CGEventTapEnable(ts_tap, true);
    return event;
  }
  CGPoint p = CGEventGetLocation(event); // already top-left-origin global coords
  CGEventFlags flags = CGEventGetFlags(event);
  int modThirds = (flags & (kCGEventFlagMaskAlternate | kCGEventFlagMaskCommand)) != 0;
  int kind = -1;
  if (type == kCGEventLeftMouseDown) kind = 0;
  else if (type == kCGEventLeftMouseDragged) kind = 1;
  else if (type == kCGEventLeftMouseUp) kind = 2;
  if (kind >= 0) goDragEvent(kind, p.x, p.y, modThirds);
  return event; // listen-only: never swallow
}

static bool ts_tap_start(void) {
  if (ts_tap) return true;
  CGEventMask mask = CGEventMaskBit(kCGEventLeftMouseDown) | CGEventMaskBit(kCGEventLeftMouseDragged) | CGEventMaskBit(kCGEventLeftMouseUp);
  ts_tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionListenOnly, mask, ts_tap_cb, NULL);
  if (!ts_tap) return false;
  ts_tap_source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, ts_tap, 0);
  CFRunLoopAddSource(CFRunLoopGetMain(), ts_tap_source, kCFRunLoopCommonModes);
  CGEventTapEnable(ts_tap, true);
  return true;
}

static void ts_tap_stop(void) {
  if (!ts_tap) return;
  CGEventTapEnable(ts_tap, false);
  CFRunLoopRemoveSource(CFRunLoopGetMain(), ts_tap_source, kCFRunLoopCommonModes);
  CFRelease(ts_tap_source); ts_tap_source = NULL;
  CFRelease(ts_tap); ts_tap = NULL;
}
