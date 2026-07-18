package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Hotkey struct {
	KeyCode   uint32 `json:"keyCode"`
	Modifiers uint32 `json:"modifiers"`
}

type Zone string

const (
	ZoneTopLeft     Zone = "top-left"
	ZoneTop         Zone = "top"
	ZoneTopRight    Zone = "top-right"
	ZoneLeft        Zone = "left"
	ZoneRight       Zone = "right"
	ZoneBottomLeft  Zone = "bottom-left"
	ZoneBottom      Zone = "bottom"
	ZoneBottomRight Zone = "bottom-right"
)

var AllZones = []Zone{ZoneTopLeft, ZoneTop, ZoneTopRight, ZoneLeft, ZoneRight, ZoneBottomLeft, ZoneBottom, ZoneBottomRight}

type GeneralSettings struct {
	LaunchAtLogin    bool    `json:"launchAtLogin"`
	ShowMenuBarIcon  bool    `json:"showMenuBarIcon"`
	EnableAnimations bool    `json:"enableAnimations"`
	WindowPadding    float64 `json:"windowPadding"`
	PadFullscreen    bool    `json:"padFullscreen"`
}

type HotkeySettings struct {
	Enabled  bool              `json:"enabled"`
	Bindings map[Action]Hotkey `json:"bindings"`
}

type SnapSettings struct {
	Enabled             bool            `json:"enabled"`
	RestorePreviousSize bool            `json:"restorePreviousSize"`
	Zones               map[Zone]Action `json:"zones"`
	ZoneThickness       float64         `json:"zoneThickness"`
	ActivationDelayMs   int             `json:"activationDelayMs"`
}

type Settings struct {
	General GeneralSettings `json:"general"`
	Hotkeys HotkeySettings  `json:"hotkeys"`
	Snap    SnapSettings    `json:"snap"`
}

const (
	modCmd  = 256
	modShift = 512
	modOpt  = 2048
	modCtrl = 4096
	keyC, keyF, keyU, keyI, keyJ, keyK, keyY = 8, 3, 32, 34, 38, 40, 16
	keyLeft, keyRight, keyDown, keyUp        = 123, 124, 125, 126
)

func DefaultSettings() Settings {
	return Settings{
		General: GeneralSettings{LaunchAtLogin: true, ShowMenuBarIcon: true, EnableAnimations: true},
		Hotkeys: HotkeySettings{
			Enabled: true,
			Bindings: map[Action]Hotkey{
				ActionCenter:          {keyC, modCmd + modOpt},
				ActionFullscreen:      {keyF, modCmd + modOpt},
				ActionHalfLeft:        {keyLeft, modCmd + modOpt},
				ActionHalfRight:       {keyRight, modCmd + modOpt},
				ActionHalfTop:         {keyUp, modCmd + modOpt},
				ActionHalfBottom:      {keyDown, modCmd + modOpt},
				ActionUpperLeft:       {keyU, modCmd + modOpt},
				ActionUpperRight:      {keyI, modCmd + modOpt},
				ActionLowerLeft:       {keyJ, modCmd + modOpt},
				ActionLowerRight:      {keyK, modCmd + modOpt},
				ActionNextThird:       {keyRight, modCtrl + modOpt},
				ActionPrevThird:       {keyLeft, modCtrl + modOpt},
				ActionTwoThirdsLeft:   {keyLeft, modCtrl + modCmd},
				ActionTwoThirdsRight:  {keyRight, modCtrl + modCmd},
				ActionTwoThirdsCenter: {keyUp, modCtrl + modCmd},
				ActionNextDisplay:     {keyRight, modCmd + modOpt + modCtrl},
				ActionPrevDisplay:     {keyLeft, modCmd + modOpt + modCtrl},
				// "undo" is a pseudo-action: it has no engine.Action const because
				// it isn't a window-layout action, only a hotkey-bindable command.
				Action("undo"): {keyY, modCmd + modOpt},
			},
		},
		Snap: SnapSettings{
			Enabled:             true,
			RestorePreviousSize: true,
			Zones: map[Zone]Action{
				ZoneTopLeft: ActionUpperLeft, ZoneTop: ActionFullscreen, ZoneTopRight: ActionUpperRight,
				ZoneLeft: ActionHalfLeft, ZoneRight: ActionHalfRight,
				ZoneBottomLeft: ActionLowerLeft, ZoneBottom: ActionNone, ZoneBottomRight: ActionLowerRight,
			},
			ZoneThickness:     10,
			ActivationDelayMs: 0,
		},
	}
}

// LoadSettings reads settings from path. Missing file → defaults. Corrupt
// file → renamed to path+".bak" and defaults returned (never crash on bad input).
func LoadSettings(path string) Settings {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultSettings()
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		_ = os.Rename(path, path+".bak")
		return DefaultSettings()
	}
	// Backfill zero-value maps from a partial file.
	if s.Hotkeys.Bindings == nil {
		s.Hotkeys.Bindings = DefaultSettings().Hotkeys.Bindings
	}
	if s.Snap.Zones == nil {
		s.Snap.Zones = DefaultSettings().Snap.Zones
	}
	return s
}

// Save writes settings atomically (tmp file + rename), creating parent dirs.
func (s Settings) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
