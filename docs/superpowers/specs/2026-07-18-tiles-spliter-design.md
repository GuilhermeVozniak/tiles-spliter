# Tiles Spliter — Design Spec

**Date:** 2026-07-18
**Status:** Approved (pending final user review)

## Overview

Tiles Spliter is a macOS menu-bar window manager with full feature parity with Sempliva's **Tiles** (closed-source freeware; behavior reverse-engineered from the app's UI/screenshots, techniques referenced from the MIT-licensed Rectangle project). Built as a **Wails v3 (alpha)** desktop app with a Go + cgo native engine and a React Preferences UI, in a **Turbo + Bun monorepo** that also contains a **Next.js 15 marketing site**.

- **Product name:** Tiles Spliter
- **Bundle ID:** `com.guilhermevozniak.tilespliter`
- **Distribution:** signed + notarized DMG (Developer ID, hardened runtime, `notarytool`), download linked from the marketing site / GitHub Releases
- **Preferences UI:** modern custom design (Tailwind 4), not a native-macOS lookalike

## Architecture (Approach A — single app, Go/cgo engine)

One Wails v3 process:

- `internal/engine` — **pure Go, zero cgo**: all geometry, snapping, undo, and settings logic. Fully unit-testable.
- `internal/platform` — thin cgo/Objective-C bridge (darwin-only): AX window access, event tap, hotkeys, overlay window, displays, permissions, login item.
- Wails v3 supplies: systray + native menu, the Preferences window (React), service bindings, and events.
- The snap-preview overlay is a **native borderless NSWindow** (translucent), never a webview.

### Monorepo layout

```
tiles-spliter/
├── apps/
│   ├── desktop/                  # Wails v3 app (Go 1.24+)
│   │   ├── main.go               # bootstrap: systray, windows, services, single-instance lock
│   │   ├── internal/
│   │   │   ├── engine/           # pure-Go logic
│   │   │   │   ├── layout.go     # frame math for all 17 actions
│   │   │   │   ├── snap.go       # snap-zone hit-testing, per-edge config, thirds modifier
│   │   │   │   ├── undo.go       # undo stack
│   │   │   │   └── settings.go   # settings model, JSON persistence, defaults, migration
│   │   │   └── platform/         # cgo/ObjC bridge (darwin only)
│   │   │       ├── ax_darwin.go/.h       # AXUIElement get/set frame of other apps' windows
│   │   │       ├── drag_darwin.go/.h     # CGEventTap drag watching + modifier keys
│   │   │       ├── hotkeys_darwin.go/.h  # Carbon RegisterEventHotKey
│   │   │       ├── overlay_darwin.go/.h  # translucent snap-preview NSWindow
│   │   │       ├── displays_darwin.go/.h # NSScreen enumeration, visible frames
│   │   │       └── system_darwin.go/.h   # AX permission, login item (SMAppService), animations
│   │   ├── frontend/             # React 19 + TS + Tailwind 4 + Vite — Preferences window only
│   │   └── build/                # icons, Info.plist, entitlements, DMG config
│   └── web/                      # Next.js 15 + React 19 + Tailwind 4 marketing site (static export)
├── packages/
│   └── shared/                   # TS types: settings schema, action/zone enums, default hotkeys
├── package.json                  # Bun workspaces
└── turbo.json                    # dev / build / lint / test pipeline
```

## Feature parity — engine

### Actions (17 + Undo)

Center, Fullscreen (maximize to visible frame, not native fullscreen), Half Left, Half Right, Half Top, Half Bottom, Upper Left, Upper Right, Lower Left, Lower Right, Next Third, Previous Third, Two Thirds Left, Two Thirds Right, Two Thirds Center, Next Display, Previous Display, Undo.

Every action: read the focused window's `AXUIElement` → compute target frame from the display's **visible frame** (menu bar/Dock excluded) minus **window padding** → push the old frame onto the undo stack → set position + size. "Enable animations" interpolates frames over ~150 ms; off = instant. Padding optionally applies to Fullscreen ("Enable for fullscreen windows").

**Thirds cycling:** Next Third detects which third the window currently occupies and cycles left → middle → right → wrap; Previous reverses. Two Thirds Left/Right/Center place the window over two of the three columns.

**Multi-display:** all math is per-`NSScreen`. Next/Previous Display moves the window to the adjacent screen preserving relative size/position.

**Undo:** per-window stack of previous frames; ⌥⌘Y or menu restores the most recent.

### Snap to Edges

- `CGEventTap` watches left-mouse window drags.
- **8 active-edge zones** (per screenshot): upper-left corner, top edge, upper-right corner, left edge, right edge, lower-left corner, bottom edge, lower-right corner — each independently mappable to an action or disabled via per-zone dropdowns.
- Cursor enters a zone → after the configured **activation delay** (0–1000 ms, default None/0) the translucent **overlay** previews the target frame → drop applies it.
- Holding **⌥ or ⌘** while dragging to the left/right edge snaps to **thirds** instead of halves.
- **Snap zone thickness** configurable (pt from edge, default 10 pt).
- **Restore previous size:** pre-snap frame remembered; dragging a snapped window away restores it.
- Master toggle: "Snap windows when moved to the edges of the screen".

### Hotkeys

Carbon `RegisterEventHotKey`, one per action. Defaults (from screenshot):

| Action | Default | Action | Default |
|---|---|---|---|
| Center | ⌥⌘C | Next Display | ⌃⌥⌘→ |
| Fullscreen | ⌥⌘F | Previous Display | ⌃⌥⌘← |
| Half Left | ⌥⌘← | Upper Left | ⌥⌘U |
| Half Right | ⌥⌘→ | Upper Right | ⌥⌘I |
| Half Top | ⌥⌘↑ | Lower Left | ⌥⌘J |
| Half Bottom | ⌥⌘↓ | Lower Right | ⌥⌘K |
| Next Third | ⌃⌥→ | Two Thirds Left | ⌃⌘← |
| Previous Third | ⌃⌥← | Two Thirds Right | ⌃⌘→ |
| Undo | ⌥⌘Y | Two Thirds Center | ⌃⌘↑ |

Global "Enable Hotkeys" toggle; every binding remappable; "Restore Defaults…" button.

## Settings & Preferences UI

**Persistence:** JSON at `~/Library/Application Support/tiles-spliter/settings.json`, written atomically; defaults created on first run; corrupt file → backed up and regenerated.

**Model:**
- General: `launchAtLogin` (default on), `showMenuBarIcon` (default on), `enableAnimations` (default on), `windowPadding` (0–50 pt, default 0), `padFullscreen` (default off)
- Hotkeys: `hotkeysEnabled`, `bindings: {action → {keyCode, modifiers}}`
- Snap: `snapEnabled`, `restorePreviousSize`, `zones: {zoneID → action|off}`, `zoneThickness`, `activationDelay`

**Tray (Wails v3 systray):** full action menu — Preferences… (⌘,), all 17 actions, Undo, About, Quit (⌘Q). Menu items call the same engine functions as hotkeys. `showMenuBarIcon` off → app runs headless; second launch from Finder (single-instance lock) reopens Preferences.

**Preferences window:** single Wails window (~640×520, hidden until requested). Modern custom Tailwind design, dark-mode aware; tabs General / Hotkeys / Snap to Edges / About. Hotkey capture fields record keystrokes in React, validate + register in Go; conflicts → inline error, binding left unassigned. Snap-zones panel is a stylized screen mockup with per-zone dropdowns.

**Data flow:** React ↔ Go via a Wails v3 `SettingsService` (`Get()`, `Update(partial)`). Go applies side effects immediately (re-register hotkeys, restart event tap, `SMAppService` login item, show/hide tray) and emits `settings:changed` back to the UI. `packages/shared` holds TS types for the settings schema, action/zone enums, and default hotkeys — kept in lockstep with Go structs via Wails v3 binding generation; also consumed by the marketing site's hotkey cheat-sheet.

**First-run:** without Accessibility permission, an onboarding view explains and triggers `AXIsProcessTrustedWithOptions` (with system prompt) + deep-link to System Settings → Privacy & Security → Accessibility; engine polls and activates once granted.

## Marketing site (`apps/web`)

Next.js 15 + React 19 + Tailwind 4, static export. Pages: landing (hero with animated tiling demo, feature grid, hotkey cheat-sheet table driven by `packages/shared` defaults, download button → latest DMG on GitHub Releases), privacy, changelog. Biome + Playwright smoke tests, matching the other `apps/web` sites in ~/Dev/pessoal.

## Build, signing, release

- Turbo tasks: `dev` (wails3 dev + next dev), `build`, `lint`, `test`.
- Release: `wails3` production build → `codesign` (Developer ID Application, hardened runtime, entitlements) → `xcrun notarytool submit --wait` → `xcrun stapler` → `create-dmg`.
- Runs locally first (Makefile/Taskfile); GitHub Actions workflow added once signing secrets exist.
- Accessibility permission requires no entitlement. Dock-icon hiding via activation policy (LSUIElement behavior) while keeping the Preferences window able to take focus.

## Testing

- **engine (pure Go):** unit tests for frame math of all 17 actions across fixture display configs (single/dual displays, Dock + menu-bar insets, padding values), thirds-cycle detection, snap-zone hit-testing (thickness, modifiers, per-zone mapping), undo stack, settings round-trip + corrupt-file recovery.
- **platform (cgo):** kept too thin to unit test; verified by a manual smoke checklist (grant permission; snap a real app to each of the 9 zones; fire every hotkey; display hop; undo; padding on/off; animations on/off).
- **frontend:** Vitest for hotkey-capture and settings-form logic.
- **web:** Playwright smoke.

## Error handling

- No AX permission → onboarding view, engine idle, no crashes.
- Non-resizable windows (some dialogs/apps) → AX setters fail per-window; log + subtle tray feedback; never crash.
- Event tap disabled by the system (timeout) → detect `kCGEventTapDisabledByTimeout` and re-enable.
- Corrupt settings → back up bad file, regenerate defaults.
- Hotkey registration conflict → inline UI error, binding unassigned.

## Out of scope (v1)

- Windows/Linux support (macOS-only, like the original).
- Homebrew cask (may follow later).
- Layout presets / custom grids beyond Tiles parity.
- Auto-update (Sparkle etc.) — DMG downloads only for v1.
