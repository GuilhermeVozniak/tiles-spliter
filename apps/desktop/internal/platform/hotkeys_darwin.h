#import <Carbon/Carbon.h>
#import <dispatch/dispatch.h>
#import <pthread.h>

extern void goHotkeyFired(unsigned int id); // exported from Go

// Carbon hotkey APIs are main-thread-affine: install/register/unregister must
// run on the main thread. ts_hk_on_main funnels every call there
// synchronously. When the caller is already on the main thread the block runs
// directly — dispatch_sync onto the current queue would deadlock.
static void ts_hk_on_main(dispatch_block_t block) {
  if (pthread_main_np() != 0) {
    block();
    return;
  }
  dispatch_sync(dispatch_get_main_queue(), block);
}

static OSStatus ts_hk_handler(EventHandlerCallRef next, EventRef evt, void *data) {
  EventHotKeyID hk;
  GetEventParameter(evt, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(hk), NULL, &hk);
  goHotkeyFired(hk.id);
  return noErr;
}

// InstallEventHandler must run once; re-installing leaks handlers.
// ts_hk_installed is only ever read or written on the main thread (every
// mutation is funneled through ts_hk_on_main), which makes the guard
// effectively atomic without an explicit lock.
static int ts_hk_installed = 0;

static void ts_hk_install(void) {
  ts_hk_on_main(^{
    if (ts_hk_installed) return;
    ts_hk_installed = 1;
    EventTypeSpec spec = {kEventClassKeyboard, kEventHotKeyPressed};
    InstallEventHandler(GetEventDispatcherTarget(), ts_hk_handler, 1, &spec, NULL, NULL);
  });
}

// Returns an opaque EventHotKeyRef, or NULL if registration failed (e.g. conflict).
static void *ts_hk_register(unsigned int id, unsigned int keycode, unsigned int mods) {
  __block EventHotKeyRef ref = NULL;
  ts_hk_on_main(^{
    EventHotKeyID hkid = {'TSPL', id};
    EventHotKeyRef r = NULL;
    if (RegisterEventHotKey(keycode, mods, hkid, GetEventDispatcherTarget(), 0, &r) == noErr) ref = r;
  });
  return ref;
}

static void ts_hk_unregister(void *ref) {
  ts_hk_on_main(^{
    UnregisterEventHotKey((EventHotKeyRef)ref);
  });
}
