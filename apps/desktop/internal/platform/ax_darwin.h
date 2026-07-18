#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>

// Private but ubiquitous (Rectangle & co. rely on it): CGWindowID for an AX window.
extern AXError _AXUIElementGetWindow(AXUIElementRef element, CGWindowID *out);

typedef struct { double x, y, w, h; int ok; } TSFrame;

static bool ts_ax_trusted(bool prompt) {
  NSDictionary *opts = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt : @(prompt)};
  return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
}

// Returns a retained AXUIElementRef for the frontmost app's focused window (NULL on failure). Caller must CFRelease.
static AXUIElementRef ts_focused_window(void) {
  NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
  if (!app) return NULL;
  AXUIElementRef appRef = AXUIElementCreateApplication(app.processIdentifier);
  if (!appRef) return NULL;
  AXUIElementRef win = NULL;
  AXUIElementCopyAttributeValue(appRef, kAXFocusedWindowAttribute, (CFTypeRef *)&win);
  CFRelease(appRef);
  return win;
}

// Returns a retained AX window element under the given top-left-origin screen point. Caller must CFRelease.
static AXUIElementRef ts_window_at(double x, double y) {
  AXUIElementRef sys = AXUIElementCreateSystemWide();
  if (!sys) return NULL;
  AXUIElementRef el = NULL;
  AXUIElementCopyElementAtPosition(sys, (float)x, (float)y, &el);
  CFRelease(sys);
  if (!el) return NULL;
  // Walk up to the containing window.
  CFStringRef role = NULL;
  if (AXUIElementCopyAttributeValue(el, kAXRoleAttribute, (CFTypeRef *)&role) == kAXErrorSuccess && role) {
    bool isWin = CFEqual(role, kAXWindowRole);
    CFRelease(role);
    if (isWin) return el;
  }
  AXUIElementRef win = NULL;
  AXUIElementCopyAttributeValue(el, kAXWindowAttribute, (CFTypeRef *)&win);
  CFRelease(el);
  return win;
}

static unsigned int ts_window_id(AXUIElementRef win) {
  CGWindowID wid = 0;
  _AXUIElementGetWindow(win, &wid);
  return (unsigned int)wid;
}

static TSFrame ts_window_frame(AXUIElementRef win) {
  TSFrame f = {0};
  AXValueRef posVal = NULL, sizeVal = NULL;
  if (AXUIElementCopyAttributeValue(win, kAXPositionAttribute, (CFTypeRef *)&posVal) != kAXErrorSuccess) return f;
  if (AXUIElementCopyAttributeValue(win, kAXSizeAttribute, (CFTypeRef *)&sizeVal) != kAXErrorSuccess) { CFRelease(posVal); return f; }
  CGPoint p; CGSize s;
  AXValueGetValue(posVal, kAXValueTypeCGPoint, &p);
  AXValueGetValue(sizeVal, kAXValueTypeCGSize, &s);
  CFRelease(posVal); CFRelease(sizeVal);
  f.x = p.x; f.y = p.y; f.w = s.width; f.h = s.height; f.ok = 1;
  return f;
}

static bool ts_set_window_frame(AXUIElementRef win, double x, double y, double w, double h) {
  CGPoint p = CGPointMake(x, y);
  CGSize s = CGSizeMake(w, h);
  AXValueRef posVal = AXValueCreate(kAXValueTypeCGPoint, &p);
  AXValueRef sizeVal = AXValueCreate(kAXValueTypeCGSize, &s);
  // Size → position → size again: handles windows with min-size constraints
  // and apps that reposition on resize.
  AXError e1 = AXUIElementSetAttributeValue(win, kAXSizeAttribute, sizeVal);
  AXError e2 = AXUIElementSetAttributeValue(win, kAXPositionAttribute, posVal);
  AXError e3 = AXUIElementSetAttributeValue(win, kAXSizeAttribute, sizeVal);
  CFRelease(posVal); CFRelease(sizeVal);
  return e1 == kAXErrorSuccess && e2 == kAXErrorSuccess && e3 == kAXErrorSuccess;
}

// Single size+position set (2 AX round-trips instead of 3). Meant for
// intermediate animation frames, where the next step overwrites any drift
// from min-size constraints; final placement should use ts_set_window_frame.
static bool ts_set_window_frame_fast(AXUIElementRef win, double x, double y, double w, double h) {
  CGPoint p = CGPointMake(x, y);
  CGSize s = CGSizeMake(w, h);
  AXValueRef posVal = AXValueCreate(kAXValueTypeCGPoint, &p);
  AXValueRef sizeVal = AXValueCreate(kAXValueTypeCGSize, &s);
  AXError e1 = AXUIElementSetAttributeValue(win, kAXSizeAttribute, sizeVal);
  AXError e2 = AXUIElementSetAttributeValue(win, kAXPositionAttribute, posVal);
  CFRelease(posVal); CFRelease(sizeVal);
  return e1 == kAXErrorSuccess && e2 == kAXErrorSuccess;
}

static void ts_release(AXUIElementRef ref) { if (ref) CFRelease(ref); }
