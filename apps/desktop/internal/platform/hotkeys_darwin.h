#import <Carbon/Carbon.h>

extern void goHotkeyFired(unsigned int id); // exported from Go

static OSStatus ts_hk_handler(EventHandlerCallRef next, EventRef evt, void *data) {
  EventHotKeyID hk;
  GetEventParameter(evt, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(hk), NULL, &hk);
  goHotkeyFired(hk.id);
  return noErr;
}

static void ts_hk_install(void) {
  static int installed = 0; // InstallEventHandler must run once; re-installing leaks handlers
  if (installed) return;
  installed = 1;
  EventTypeSpec spec = {kEventClassKeyboard, kEventHotKeyPressed};
  InstallEventHandler(GetEventDispatcherTarget(), ts_hk_handler, 1, &spec, NULL, NULL);
}

// Returns an opaque EventHotKeyRef, or NULL if registration failed (e.g. conflict).
static void *ts_hk_register(unsigned int id, unsigned int keycode, unsigned int mods) {
  EventHotKeyID hkid = {'TSPL', id};
  EventHotKeyRef ref = NULL;
  if (RegisterEventHotKey(keycode, mods, hkid, GetEventDispatcherTarget(), 0, &ref) != noErr) return NULL;
  return ref;
}

static void ts_hk_unregister(void *ref) { UnregisterEventHotKey((EventHotKeyRef)ref); }
