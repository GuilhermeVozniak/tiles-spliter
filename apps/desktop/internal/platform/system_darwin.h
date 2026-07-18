#import <Cocoa/Cocoa.h>
#import <ServiceManagement/ServiceManagement.h>

// Returns NULL on success, else an error description (caller-owned C string).
static const char *ts_login_item(bool enable) {
  if (@available(macOS 13.0, *)) {
    NSError *err = nil;
    SMAppService *svc = [SMAppService mainAppService];
    BOOL ok = enable ? [svc registerAndReturnError:&err] : [svc unregisterAndReturnError:&err];
    if (ok || !err) return NULL;
    return strdup(err.localizedDescription.UTF8String);
  }
  return strdup("macOS 13+ required");
}

static void ts_activate_prefs(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    [NSApp activateIgnoringOtherApps:YES];
  });
}

static void ts_hide_from_dock(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
  });
}
