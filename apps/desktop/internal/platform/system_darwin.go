//go:build darwin

package platform

/*
#cgo LDFLAGS: -framework ServiceManagement
#include <stdlib.h>
#include "system_darwin.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

// SetLoginItem registers/unregisters the app as a login item (macOS 13+).
func SetLoginItem(enabled bool) error {
	msg := C.ts_login_item(C.bool(enabled))
	if msg == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(msg))
	return errors.New("platform: login item: " + C.GoString(msg))
}

func ActivatePrefs() { C.ts_activate_prefs() }
func HideFromDock()  { C.ts_hide_from_dock() }
