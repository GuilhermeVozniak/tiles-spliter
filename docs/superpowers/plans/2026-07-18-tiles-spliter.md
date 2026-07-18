# Tiles Spliter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Full-feature-parity clone of the macOS "Tiles" window manager (menu-bar app: 17 window actions + undo, snap-to-edges with configurable zones, remappable global hotkeys, multi-display) plus a marketing website, in one monorepo.

**Architecture:** Single Wails v3 (alpha) process. `internal/engine` is pure Go (geometry, snapping state machine, undo, settings) and fully unit-tested. `internal/platform` is a thin cgo/Objective-C bridge (AX API, CGEventTap, Carbon hotkeys, overlay NSPanel, NSScreen, SMAppService). React 19 is used only for the Preferences window; Next.js 15 for the marketing site.

**Tech Stack:** Go 1.24+, Wails v3 alpha (`github.com/wailsapp/wails/v3`), cgo/Objective-C (Cocoa, ApplicationServices, Carbon, ServiceManagement), React 19 + TypeScript + Tailwind 4 + Vite, Next.js 15, Bun workspaces + Turbo, Biome, Vitest, Playwright.

## Global Constraints

- Product name: **Tiles Spliter**; bundle ID: **`com.guilhermevozniak.tilespliter`**.
- macOS 13.0+ minimum (required for `SMAppService` login items). Universal binary (arm64 + x86_64) at release time.
- Wails **v3 alpha** — install CLI with `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`. The alpha API drifts: if a name in this plan fails to compile, check `go doc github.com/wailsapp/wails/v3/pkg/application <Symbol>` and adapt the name only, not the design.
- `internal/engine` must stay **pure Go — zero cgo imports**. All cgo lives in `internal/platform` behind `//go:build darwin`.
- All Go-side geometry uses **top-left-origin global coordinates** (y grows downward), matching the AX API. Only `internal/platform` converts to/from Cocoa's bottom-left origin.
- Monorepo: Bun workspaces + Turbo, layout `apps/desktop`, `apps/web`, `packages/shared` (same shape as the sibling repos app-cleaner / option-tab / drag-zone).
- TS tooling: React 19, TypeScript strict, Tailwind 4, Vite (desktop frontend), Next.js 15 static export (web), Biome for lint/format, Vitest for unit tests.
- TDD for everything in `internal/engine` and frontend logic. cgo code is verified via the `cmd/axprobe` manual probe CLI + smoke checklist (Task 24).
- Commit after every task (small, working commits). Spec: `docs/superpowers/specs/2026-07-18-tiles-spliter-design.md`.
- Settings file path: `~/Library/Application Support/tiles-spliter/settings.json`.

---

### Task 1: Monorepo scaffold

**Files:**
- Create: `package.json`, `turbo.json`, `biome.json`, `.gitignore`, `README.md`

**Interfaces:**
- Produces: Bun workspaces (`apps/*`, `packages/*`), Turbo tasks `dev`, `build`, `test`, `lint` that later tasks plug into.

- [ ] **Step 1: Write root config files**

`package.json`:
```json
{
  "name": "tiles-spliter",
  "private": true,
  "workspaces": ["apps/*", "packages/*", "apps/desktop/frontend"],
  "packageManager": "bun@1.2.0",
  "scripts": {
    "dev": "turbo dev",
    "build": "turbo build",
    "test": "turbo test",
    "lint": "turbo lint"
  },
  "devDependencies": {
    "@biomejs/biome": "^2.0.0",
    "turbo": "^2.5.0"
  }
}
```

`turbo.json`:
```json
{
  "$schema": "https://turbo.build/schema.json",
  "tasks": {
    "build": { "dependsOn": ["^build"], "outputs": ["dist/**", ".next/**", "out/**", "bin/**"] },
    "test": { "dependsOn": ["^build"] },
    "lint": {},
    "dev": { "cache": false, "persistent": true }
  }
}
```

`biome.json`:
```json
{
  "$schema": "https://biomejs.dev/schemas/2.0.0/schema.json",
  "formatter": { "enabled": true, "indentStyle": "space", "indentWidth": 2 },
  "linter": { "enabled": true, "rules": { "recommended": true } },
  "javascript": { "formatter": { "quoteStyle": "double" } }
}
```

`.gitignore`:
```
node_modules/
dist/
out/
.next/
.turbo/
bin/
apps/desktop/frontend/bindings/
apps/desktop/build/darwin/Tiles Spliter.app
*.dmg
.DS_Store
```

`README.md`: one paragraph describing the app + `bun install && bun run dev` quickstart.

- [ ] **Step 2: Verify workspace resolves**

Run: `bun install && bunx turbo --version`
Expected: install succeeds (lockfile created), turbo prints a 2.x version.

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "chore: scaffold Turbo + Bun monorepo"
```

---

### Task 2: `packages/shared` — settings schema, actions, defaults

**Files:**
- Create: `packages/shared/package.json`, `packages/shared/src/index.ts`, `packages/shared/src/index.test.ts`, `packages/shared/tsconfig.json`, `packages/shared/vitest.config.ts`

**Interfaces:**
- Produces (imported by desktop frontend and web as `@tiles-spliter/shared`):
  - `type Action` — union of the 17 action ids + `"none"`
  - `type ZoneID` — 8 zone ids
  - `interface Hotkey { keyCode: number; modifiers: number }` (Carbon codes)
  - `interface Settings { general: {...}; hotkeys: {...}; snap: {...} }` mirroring the Go struct JSON exactly
  - `const DEFAULT_SETTINGS: Settings`, `const ACTION_LABELS: Record<Action, string>`, `const HOTKEY_DISPLAY: (hk: Hotkey) => string`

- [ ] **Step 1: Write failing test**

`packages/shared/src/index.test.ts`:
```ts
import { describe, expect, it } from "vitest";
import { ACTION_LABELS, ACTIONS, DEFAULT_SETTINGS, HOTKEY_DISPLAY, ZONES } from "./index";

describe("shared defaults", () => {
  it("has 17 actions plus none", () => {
    expect(ACTIONS).toHaveLength(18);
    expect(ACTIONS).toContain("center");
    expect(ACTIONS).toContain("none");
  });
  it("has 8 zones with parity defaults", () => {
    expect(ZONES).toHaveLength(8);
    expect(DEFAULT_SETTINGS.snap.zones["left"]).toBe("half-left");
    expect(DEFAULT_SETTINGS.snap.zones["top"]).toBe("fullscreen");
    expect(DEFAULT_SETTINGS.snap.zones["bottom"]).toBe("none");
    expect(DEFAULT_SETTINGS.snap.zoneThickness).toBe(10);
    expect(DEFAULT_SETTINGS.snap.activationDelayMs).toBe(0);
  });
  it("default hotkeys match Tiles (spot checks)", () => {
    // Carbon: cmd=256, opt=2048, ctrl=4096; C=8, left arrow=123
    expect(DEFAULT_SETTINGS.hotkeys.bindings["center"]).toEqual({ keyCode: 8, modifiers: 256 + 2048 });
    expect(DEFAULT_SETTINGS.hotkeys.bindings["half-left"]).toEqual({ keyCode: 123, modifiers: 256 + 2048 });
    expect(DEFAULT_SETTINGS.hotkeys.bindings["next-display"]).toEqual({ keyCode: 124, modifiers: 256 + 2048 + 4096 });
  });
  it("formats hotkeys for display", () => {
    expect(HOTKEY_DISPLAY({ keyCode: 8, modifiers: 256 + 2048 })).toBe("⌥⌘C");
  });
  it("labels every action", () => {
    for (const a of ACTIONS) expect(ACTION_LABELS[a]).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd packages/shared && bunx vitest run`
Expected: FAIL (cannot resolve `./index`).

- [ ] **Step 3: Implement**

`packages/shared/package.json`:
```json
{
  "name": "@tiles-spliter/shared",
  "version": "0.1.0",
  "type": "module",
  "main": "src/index.ts",
  "types": "src/index.ts",
  "scripts": { "test": "vitest run", "lint": "biome check src" },
  "devDependencies": { "typescript": "^5.8.0", "vitest": "^3.0.0" }
}
```

`packages/shared/src/index.ts`:
```ts
export const ACTIONS = [
  "center", "fullscreen",
  "half-left", "half-right", "half-top", "half-bottom",
  "upper-left", "upper-right", "lower-left", "lower-right",
  "next-third", "prev-third",
  "two-thirds-left", "two-thirds-right", "two-thirds-center",
  "next-display", "prev-display",
  "none",
] as const;
export type Action = (typeof ACTIONS)[number];

export const ZONES = [
  "top-left", "top", "top-right",
  "left", "right",
  "bottom-left", "bottom", "bottom-right",
] as const;
export type ZoneID = (typeof ZONES)[number];

export interface Hotkey { keyCode: number; modifiers: number }

export interface Settings {
  general: {
    launchAtLogin: boolean;
    showMenuBarIcon: boolean;
    enableAnimations: boolean;
    windowPadding: number;
    padFullscreen: boolean;
  };
  hotkeys: {
    enabled: boolean;
    bindings: Partial<Record<Action, Hotkey>>;
  };
  snap: {
    enabled: boolean;
    restorePreviousSize: boolean;
    zones: Record<ZoneID, Action>;
    zoneThickness: number;
    activationDelayMs: number;
  };
}

// Carbon modifier masks / virtual key codes
export const MOD = { cmd: 256, shift: 512, opt: 2048, ctrl: 4096 } as const;
export const KEY = {
  C: 8, F: 3, U: 32, I: 34, J: 38, K: 40, Y: 16,
  left: 123, right: 124, down: 125, up: 126,
} as const;

const b = (keyCode: number, modifiers: number): Hotkey => ({ keyCode, modifiers });

export const DEFAULT_SETTINGS: Settings = {
  general: { launchAtLogin: true, showMenuBarIcon: true, enableAnimations: true, windowPadding: 0, padFullscreen: false },
  hotkeys: {
    enabled: true,
    bindings: {
      center: b(KEY.C, MOD.cmd + MOD.opt),
      fullscreen: b(KEY.F, MOD.cmd + MOD.opt),
      "half-left": b(KEY.left, MOD.cmd + MOD.opt),
      "half-right": b(KEY.right, MOD.cmd + MOD.opt),
      "half-top": b(KEY.up, MOD.cmd + MOD.opt),
      "half-bottom": b(KEY.down, MOD.cmd + MOD.opt),
      "upper-left": b(KEY.U, MOD.cmd + MOD.opt),
      "upper-right": b(KEY.I, MOD.cmd + MOD.opt),
      "lower-left": b(KEY.J, MOD.cmd + MOD.opt),
      "lower-right": b(KEY.K, MOD.cmd + MOD.opt),
      "next-third": b(KEY.right, MOD.ctrl + MOD.opt),
      "prev-third": b(KEY.left, MOD.ctrl + MOD.opt),
      "two-thirds-left": b(KEY.left, MOD.ctrl + MOD.cmd),
      "two-thirds-right": b(KEY.right, MOD.ctrl + MOD.cmd),
      "two-thirds-center": b(KEY.up, MOD.ctrl + MOD.cmd),
      "next-display": b(KEY.right, MOD.cmd + MOD.opt + MOD.ctrl),
      "prev-display": b(KEY.left, MOD.cmd + MOD.opt + MOD.ctrl),
    },
  },
  snap: {
    enabled: true,
    restorePreviousSize: true,
    zones: {
      "top-left": "upper-left", top: "fullscreen", "top-right": "upper-right",
      left: "half-left", right: "half-right",
      "bottom-left": "lower-left", bottom: "none", "bottom-right": "lower-right",
    },
    zoneThickness: 10,
    activationDelayMs: 0,
  },
};

export const ACTION_LABELS: Record<Action, string> = {
  center: "Center", fullscreen: "Fullscreen",
  "half-left": "Half Left", "half-right": "Half Right", "half-top": "Half Top", "half-bottom": "Half Bottom",
  "upper-left": "Upper Left", "upper-right": "Upper Right", "lower-left": "Lower Left", "lower-right": "Lower Right",
  "next-third": "Next Third", "prev-third": "Previous Third",
  "two-thirds-left": "Two Thirds Left", "two-thirds-right": "Two Thirds Right", "two-thirds-center": "Two Thirds Center",
  "next-display": "Next Display", "prev-display": "Previous Display",
  none: "—",
};

const KEY_NAMES: Record<number, string> = {
  8: "C", 3: "F", 32: "U", 34: "I", 38: "J", 40: "K", 16: "Y",
  123: "←", 124: "→", 125: "↓", 126: "↑",
};

export function HOTKEY_DISPLAY(hk: Hotkey): string {
  let s = "";
  if (hk.modifiers & MOD.ctrl) s += "⌃";
  if (hk.modifiers & MOD.opt) s += "⌥";
  if (hk.modifiers & MOD.shift) s += "⇧";
  if (hk.modifiers & MOD.cmd) s += "⌘";
  return s + (KEY_NAMES[hk.keyCode] ?? `#${hk.keyCode}`);
}
```

`packages/shared/tsconfig.json`: strict, `"module": "ESNext"`, `"moduleResolution": "bundler"`, `"target": "ES2022"`.
`packages/shared/vitest.config.ts`: `import { defineConfig } from "vitest/config"; export default defineConfig({});`

- [ ] **Step 4: Run test to verify it passes**

Run: `cd packages/shared && bun install && bunx vitest run`
Expected: 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(shared): settings schema, actions, Tiles-parity defaults"
```

---

### Task 3: Engine — geometry types + layout math (halves, quarters, center, fullscreen, two-thirds, padding)

**Files:**
- Create: `apps/desktop/go.mod`, `apps/desktop/internal/engine/geometry.go`, `apps/desktop/internal/engine/layout.go`, `apps/desktop/internal/engine/layout_test.go`

**Interfaces:**
- Produces (used by every later Go task):
  - `type Rect struct { X, Y, W, H float64 }` with `func (r Rect) Contains(p Point) bool`, `func (r Rect) Mid() Point`, `func (r Rect) Eq(o Rect, eps float64) bool`
  - `type Point struct { X, Y float64 }`
  - `type Display struct { ID uint32; Frame, Visible Rect }` — top-left-origin coords
  - `type Action string` + constants `ActionCenter`, `ActionFullscreen`, `ActionHalfLeft`, `ActionHalfRight`, `ActionHalfTop`, `ActionHalfBottom`, `ActionUpperLeft`, `ActionUpperRight`, `ActionLowerLeft`, `ActionLowerRight`, `ActionNextThird`, `ActionPrevThird`, `ActionTwoThirdsLeft`, `ActionTwoThirdsRight`, `ActionTwoThirdsCenter`, `ActionNextDisplay`, `ActionPrevDisplay`, `ActionNone` — string values identical to `packages/shared` (`"center"`, `"half-left"`, …)
  - `func FrameFor(a Action, win Rect, d Display, pad float64, padFullscreen bool) (Rect, bool)` — returns false for actions it doesn't handle (thirds/display actions, Tasks 4–5)

- [ ] **Step 1: Init Go module**

Run: `mkdir -p apps/desktop && cd apps/desktop && go mod init github.com/GuilhermeVozniak/tiles-spliter/desktop`
Expected: `go.mod` with `go 1.24`.

- [ ] **Step 2: Write failing tests**

`apps/desktop/internal/engine/layout_test.go`:
```go
package engine

import "testing"

// 1440x900 display, 25pt menu bar, 60pt Dock at bottom.
var disp = Display{ID: 1, Frame: Rect{0, 0, 1440, 900}, Visible: Rect{0, 25, 1440, 815}}

func eq(t *testing.T, got, want Rect) {
	t.Helper()
	if !got.Eq(want, 0.01) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func must(t *testing.T, r Rect, ok bool) Rect {
	t.Helper()
	if !ok {
		t.Fatal("expected handled action")
	}
	return r
}

func TestHalvesNoPadding(t *testing.T) {
	w := Rect{100, 100, 400, 300}
	eq(t, must(t, FrameFor(ActionHalfLeft, w, disp, 0, false)), Rect{0, 25, 720, 815})
	eq(t, must(t, FrameFor(ActionHalfRight, w, disp, 0, false)), Rect{720, 25, 720, 815})
	eq(t, must(t, FrameFor(ActionHalfTop, w, disp, 0, false)), Rect{0, 25, 1440, 407.5})
	eq(t, must(t, FrameFor(ActionHalfBottom, w, disp, 0, false)), Rect{0, 432.5, 1440, 407.5})
}

func TestQuarters(t *testing.T) {
	w := Rect{0, 0, 10, 10}
	eq(t, must(t, FrameFor(ActionUpperLeft, w, disp, 0, false)), Rect{0, 25, 720, 407.5})
	eq(t, must(t, FrameFor(ActionUpperRight, w, disp, 0, false)), Rect{720, 25, 720, 407.5})
	eq(t, must(t, FrameFor(ActionLowerLeft, w, disp, 0, false)), Rect{0, 432.5, 720, 407.5})
	eq(t, must(t, FrameFor(ActionLowerRight, w, disp, 0, false)), Rect{720, 432.5, 720, 407.5})
}

func TestPaddingOuterFullInnerHalf(t *testing.T) {
	// pad=10: outer edges inset 10, shared inner edge each side inset 5.
	got := must(t, FrameFor(ActionHalfLeft, Rect{}, disp, 10, false))
	eq(t, got, Rect{10, 35, 720 - 10 - 5, 815 - 20})
}

func TestFullscreenPadding(t *testing.T) {
	eq(t, must(t, FrameFor(ActionFullscreen, Rect{}, disp, 10, false)), disp.Visible)
	eq(t, must(t, FrameFor(ActionFullscreen, Rect{}, disp, 10, true)), Rect{10, 35, 1420, 795})
}

func TestCenterKeepsSizeAndClamps(t *testing.T) {
	eq(t, must(t, FrameFor(ActionCenter, Rect{0, 0, 400, 300}, disp, 0, false)), Rect{520, 282.5, 400, 300})
	// Oversized window is clamped to the visible frame.
	eq(t, must(t, FrameFor(ActionCenter, Rect{0, 0, 2000, 1000}, disp, 0, false)), disp.Visible)
}

func TestTwoThirds(t *testing.T) {
	eq(t, must(t, FrameFor(ActionTwoThirdsLeft, Rect{}, disp, 0, false)), Rect{0, 25, 960, 815})
	eq(t, must(t, FrameFor(ActionTwoThirdsRight, Rect{}, disp, 0, false)), Rect{480, 25, 960, 815})
	eq(t, must(t, FrameFor(ActionTwoThirdsCenter, Rect{}, disp, 0, false)), Rect{240, 25, 960, 815})
}

func TestUnhandledActionsReturnFalse(t *testing.T) {
	if _, ok := FrameFor(ActionNextThird, Rect{}, disp, 0, false); ok {
		t.Fatal("thirds are handled elsewhere")
	}
	if _, ok := FrameFor(ActionNextDisplay, Rect{}, disp, 0, false); ok {
		t.Fatal("display moves are handled elsewhere")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build (`undefined: Display`, `FrameFor`, …).

- [ ] **Step 4: Implement**

`apps/desktop/internal/engine/geometry.go`:
```go
// Package engine contains all window-management logic as pure Go.
// Coordinates are top-left-origin global points (y grows downward), matching the AX API.
package engine

import "math"

type Point struct{ X, Y float64 }

type Rect struct{ X, Y, W, H float64 }

func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

func (r Rect) Mid() Point { return Point{r.X + r.W/2, r.Y + r.H/2} }

func (r Rect) Eq(o Rect, eps float64) bool {
	return math.Abs(r.X-o.X) < eps && math.Abs(r.Y-o.Y) < eps &&
		math.Abs(r.W-o.W) < eps && math.Abs(r.H-o.H) < eps
}

// Display describes one screen. Visible excludes the menu bar and Dock.
type Display struct {
	ID             uint32
	Frame, Visible Rect
}
```

`apps/desktop/internal/engine/layout.go`:
```go
package engine

type Action string

const (
	ActionCenter          Action = "center"
	ActionFullscreen      Action = "fullscreen"
	ActionHalfLeft        Action = "half-left"
	ActionHalfRight       Action = "half-right"
	ActionHalfTop         Action = "half-top"
	ActionHalfBottom      Action = "half-bottom"
	ActionUpperLeft       Action = "upper-left"
	ActionUpperRight      Action = "upper-right"
	ActionLowerLeft       Action = "lower-left"
	ActionLowerRight      Action = "lower-right"
	ActionNextThird       Action = "next-third"
	ActionPrevThird       Action = "prev-third"
	ActionTwoThirdsLeft   Action = "two-thirds-left"
	ActionTwoThirdsRight  Action = "two-thirds-right"
	ActionTwoThirdsCenter Action = "two-thirds-center"
	ActionNextDisplay     Action = "next-display"
	ActionPrevDisplay     Action = "prev-display"
	ActionNone            Action = "none"
)

const frac = 1e-9

// cell returns the sub-rect of v at fractional origin (fx,fy) with fractional
// size (fw,fh), padded: outer edges (touching v's edge) inset by pad, inner
// shared edges inset by pad/2 so adjacent cells end up pad apart.
func cell(v Rect, fx, fy, fw, fh, pad float64) Rect {
	r := Rect{v.X + fx*v.W, v.Y + fy*v.H, fw * v.W, fh * v.H}
	left, top, right, bottom := pad/2, pad/2, pad/2, pad/2
	if fx <= frac {
		left = pad
	}
	if fy <= frac {
		top = pad
	}
	if fx+fw >= 1-frac {
		right = pad
	}
	if fy+fh >= 1-frac {
		bottom = pad
	}
	return Rect{r.X + left, r.Y + top, r.W - left - right, r.H - top - bottom}
}

// FrameFor computes the target frame for a stateless action. Thirds cycling
// (needs current position) and display moves (need the display list) are
// handled by ThirdFrame / MapToDisplay; for those it returns ok=false.
func FrameFor(a Action, win Rect, d Display, pad float64, padFullscreen bool) (Rect, bool) {
	v := d.Visible
	switch a {
	case ActionHalfLeft:
		return cell(v, 0, 0, 0.5, 1, pad), true
	case ActionHalfRight:
		return cell(v, 0.5, 0, 0.5, 1, pad), true
	case ActionHalfTop:
		return cell(v, 0, 0, 1, 0.5, pad), true
	case ActionHalfBottom:
		return cell(v, 0, 0.5, 1, 0.5, pad), true
	case ActionUpperLeft:
		return cell(v, 0, 0, 0.5, 0.5, pad), true
	case ActionUpperRight:
		return cell(v, 0.5, 0, 0.5, 0.5, pad), true
	case ActionLowerLeft:
		return cell(v, 0, 0.5, 0.5, 0.5, pad), true
	case ActionLowerRight:
		return cell(v, 0.5, 0.5, 0.5, 0.5, pad), true
	case ActionTwoThirdsLeft:
		return cell(v, 0, 0, 2.0/3, 1, pad), true
	case ActionTwoThirdsRight:
		return cell(v, 1.0/3, 0, 2.0/3, 1, pad), true
	case ActionTwoThirdsCenter:
		return cell(v, 1.0/6, 0, 2.0/3, 1, pad), true
	case ActionFullscreen:
		if padFullscreen {
			return cell(v, 0, 0, 1, 1, pad), true
		}
		return v, true
	case ActionCenter:
		w, h := min(win.W, v.W), min(win.H, v.H)
		return Rect{v.X + (v.W-w)/2, v.Y + (v.H-h)/2, w, h}, true
	}
	return Rect{}, false
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -v`
Expected: all 7 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat(engine): geometry types and layout math with padding"
```

---

### Task 4: Engine — thirds cycling

**Files:**
- Create: `apps/desktop/internal/engine/thirds.go`, `apps/desktop/internal/engine/thirds_test.go`

**Interfaces:**
- Consumes: `Rect`, `Display`, `cell`, `Action` (Task 3)
- Produces: `func ThirdFrame(a Action, win Rect, d Display, pad float64) (Rect, bool)` — handles `ActionNextThird`/`ActionPrevThird`; also `func ThirdColumn(i int, d Display, pad float64) Rect` (used by snap modifier-thirds in Task 9)

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/engine/thirds_test.go`:
```go
package engine

import "testing"

func third(i int) Rect { return ThirdColumn(i, disp, 0) }

func TestThirdColumns(t *testing.T) {
	eq(t, third(0), Rect{0, 25, 480, 815})
	eq(t, third(1), Rect{480, 25, 480, 815})
	eq(t, third(2), Rect{960, 25, 480, 815})
}

func TestNextThirdFromUnpositionedStartsLeft(t *testing.T) {
	got := must(t, ThirdFrame(ActionNextThird, Rect{100, 100, 500, 400}, disp, 0))
	eq(t, got, third(0))
}

func TestNextThirdCyclesAndWraps(t *testing.T) {
	eq(t, must(t, ThirdFrame(ActionNextThird, third(0), disp, 0)), third(1))
	eq(t, must(t, ThirdFrame(ActionNextThird, third(1), disp, 0)), third(2))
	eq(t, must(t, ThirdFrame(ActionNextThird, third(2), disp, 0)), third(0))
}

func TestPrevThirdCyclesAndWraps(t *testing.T) {
	eq(t, must(t, ThirdFrame(ActionPrevThird, Rect{100, 100, 500, 400}, disp, 0)), third(2))
	eq(t, must(t, ThirdFrame(ActionPrevThird, third(0), disp, 0)), third(2))
	eq(t, must(t, ThirdFrame(ActionPrevThird, third(2), disp, 0)), third(1))
}

func TestThirdFrameIgnoresOtherActions(t *testing.T) {
	if _, ok := ThirdFrame(ActionCenter, Rect{}, disp, 0); ok {
		t.Fatal("only thirds actions")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build (`undefined: ThirdColumn`).

- [ ] **Step 3: Implement**

`apps/desktop/internal/engine/thirds.go`:
```go
package engine

// ThirdColumn returns column i (0..2) of the display's visible frame.
func ThirdColumn(i int, d Display, pad float64) Rect {
	return cell(d.Visible, float64(i)/3, 0, 1.0/3, 1, pad)
}

// currentThird reports which third the window currently occupies, or -1.
// A window "occupies" a third when its frame matches within a tolerance —
// tolerant of padding differences by comparing against every padding=pad column.
func currentThird(win Rect, d Display, pad float64) int {
	for i := 0; i < 3; i++ {
		if win.Eq(ThirdColumn(i, d, pad), 2.0) {
			return i
		}
	}
	return -1
}

// ThirdFrame handles ActionNextThird / ActionPrevThird: cycle the window
// left → middle → right (wrapping); a window not currently on a third starts
// at the left (Next) or right (Prev) column, matching Tiles.
func ThirdFrame(a Action, win Rect, d Display, pad float64) (Rect, bool) {
	cur := currentThird(win, d, pad)
	switch a {
	case ActionNextThird:
		return ThirdColumn((cur+1)%3, d, pad), true // cur==-1 → 0
	case ActionPrevThird:
		if cur <= 0 {
			return ThirdColumn(2, d, pad), true
		}
		return ThirdColumn(cur-1, d, pad), true
	}
	return Rect{}, false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -run Third -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(engine): thirds cycling with wrap, matching Tiles"
```

---

### Task 5: Engine — multi-display moves

**Files:**
- Create: `apps/desktop/internal/engine/displays.go`, `apps/desktop/internal/engine/displays_test.go`

**Interfaces:**
- Consumes: `Rect`, `Display` (Task 3)
- Produces:
  - `func DisplayOf(win Rect, displays []Display) int` — index of the display containing the window's midpoint (falls back to 0)
  - `func AdjacentDisplay(cur int, displays []Display, next bool) int` — displays ordered left-to-right by Frame.X, wrapping
  - `func MapToDisplay(win Rect, from, to Display) Rect` — proportional re-mapping of the window into the target's visible frame

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/engine/displays_test.go`:
```go
package engine

import "testing"

var dispB = Display{ID: 2, Frame: Rect{1440, 0, 1920, 1080}, Visible: Rect{1440, 25, 1920, 1055}}
var two = []Display{disp, dispB}

func TestDisplayOf(t *testing.T) {
	if DisplayOf(Rect{100, 100, 400, 300}, two) != 0 {
		t.Fatal("want display 0")
	}
	if DisplayOf(Rect{2000, 100, 400, 300}, two) != 1 {
		t.Fatal("want display 1")
	}
	if DisplayOf(Rect{-5000, -5000, 10, 10}, two) != 0 {
		t.Fatal("off-screen falls back to 0")
	}
}

func TestAdjacentDisplayWraps(t *testing.T) {
	if AdjacentDisplay(1, two, true) != 0 {
		t.Fatal("next from last wraps to first")
	}
	if AdjacentDisplay(0, two, false) != 1 {
		t.Fatal("prev from first wraps to last")
	}
}

func TestMapToDisplayProportional(t *testing.T) {
	// Left half of A maps to left half of B.
	win := Rect{0, 25, 720, 815}
	eq(t, MapToDisplay(win, disp, dispB), Rect{1440, 25, 960, 1055})
}

func TestMapToDisplayKeepsRelativeOffset(t *testing.T) {
	// Centered quarter-size window stays centered.
	win := Rect{360, 25 + 203.75, 720, 407.5}
	got := MapToDisplay(win, disp, dispB)
	eq(t, got, Rect{1440 + 480, 25 + 263.75, 960, 527.5})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build.

- [ ] **Step 3: Implement**

`apps/desktop/internal/engine/displays.go`:
```go
package engine

import "sort"

// DisplayOf returns the index of the display whose visible frame contains the
// window's midpoint, falling back to the nearest-by-frame (index 0 last resort).
func DisplayOf(win Rect, displays []Display) int {
	mid := win.Mid()
	for i, d := range displays {
		if d.Frame.Contains(mid) {
			return i
		}
	}
	return 0
}

// AdjacentDisplay returns the next/previous display index in left-to-right
// order of Frame.X, wrapping around.
func AdjacentDisplay(cur int, displays []Display, next bool) int {
	if len(displays) < 2 {
		return cur
	}
	order := make([]int, len(displays))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return displays[order[a]].Frame.X < displays[order[b]].Frame.X
	})
	pos := 0
	for i, idx := range order {
		if idx == cur {
			pos = i
			break
		}
	}
	if next {
		return order[(pos+1)%len(order)]
	}
	return order[(pos-1+len(order))%len(order)]
}

// MapToDisplay proportionally re-maps a window from one display's visible
// frame into another's, preserving relative position and relative size.
func MapToDisplay(win Rect, from, to Display) Rect {
	f, t := from.Visible, to.Visible
	sx, sy := t.W/f.W, t.H/f.H
	return Rect{
		X: t.X + (win.X-f.X)*sx,
		Y: t.Y + (win.Y-f.Y)*sy,
		W: win.W * sx,
		H: win.H * sy,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -run Display -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(engine): multi-display detection and proportional window mapping"
```

---

### Task 6: Engine — undo stack

**Files:**
- Create: `apps/desktop/internal/engine/undo.go`, `apps/desktop/internal/engine/undo_test.go`

**Interfaces:**
- Consumes: `Rect` (Task 3)
- Produces: `type UndoStack struct` with methods `Push(winID uint32, frame Rect)`, `Pop(winID uint32) (Rect, bool)`, safe for concurrent use, capped at 20 frames per window.

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/engine/undo_test.go`:
```go
package engine

import "testing"

func TestUndoLIFOPerWindow(t *testing.T) {
	u := NewUndoStack()
	u.Push(1, Rect{1, 0, 0, 0})
	u.Push(1, Rect{2, 0, 0, 0})
	u.Push(2, Rect{9, 0, 0, 0})
	if r, ok := u.Pop(1); !ok || r.X != 2 {
		t.Fatalf("want LIFO for window 1, got %+v %v", r, ok)
	}
	if r, ok := u.Pop(2); !ok || r.X != 9 {
		t.Fatalf("windows are independent, got %+v %v", r, ok)
	}
	if _, ok := u.Pop(2); ok {
		t.Fatal("empty stack pops nothing")
	}
}

func TestUndoCap(t *testing.T) {
	u := NewUndoStack()
	for i := 0; i < 30; i++ {
		u.Push(1, Rect{X: float64(i)})
	}
	count := 0
	for {
		if _, ok := u.Pop(1); !ok {
			break
		}
		count++
	}
	if count != 20 {
		t.Fatalf("cap at 20, got %d", count)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build (`undefined: NewUndoStack`).

- [ ] **Step 3: Implement**

`apps/desktop/internal/engine/undo.go`:
```go
package engine

import "sync"

const undoCap = 20

// UndoStack remembers previous window frames per CGWindowID.
type UndoStack struct {
	mu     sync.Mutex
	frames map[uint32][]Rect
}

func NewUndoStack() *UndoStack {
	return &UndoStack{frames: map[uint32][]Rect{}}
}

func (u *UndoStack) Push(winID uint32, frame Rect) {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := append(u.frames[winID], frame)
	if len(s) > undoCap {
		s = s[len(s)-undoCap:]
	}
	u.frames[winID] = s
}

func (u *UndoStack) Pop(winID uint32) (Rect, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := u.frames[winID]
	if len(s) == 0 {
		return Rect{}, false
	}
	r := s[len(s)-1]
	u.frames[winID] = s[:len(s)-1]
	return r, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -run Undo -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(engine): per-window undo stack"
```

---

### Task 7: Engine — settings model, defaults, persistence

**Files:**
- Create: `apps/desktop/internal/engine/settings.go`, `apps/desktop/internal/engine/settings_test.go`

**Interfaces:**
- Consumes: `Action` (Task 3)
- Produces (JSON tags must serialize to exactly the same shape as `Settings` in `packages/shared`):
  - `type Hotkey struct { KeyCode uint32 `json:"keyCode"`; Modifiers uint32 `json:"modifiers"` }`
  - `type Zone string` + constants `ZoneTopLeft ("top-left")`, `ZoneTop`, `ZoneTopRight`, `ZoneLeft`, `ZoneRight`, `ZoneBottomLeft`, `ZoneBottom`, `ZoneBottomRight`
  - `type Settings struct { General GeneralSettings `json:"general"`; Hotkeys HotkeySettings `json:"hotkeys"`; Snap SnapSettings `json:"snap"` }` with nested structs mirroring the shared TS types (fields: `launchAtLogin`, `showMenuBarIcon`, `enableAnimations`, `windowPadding`, `padFullscreen`; `enabled`, `bindings`; `enabled`, `restorePreviousSize`, `zones`, `zoneThickness`, `activationDelayMs`)
  - `func DefaultSettings() Settings` — same values as `DEFAULT_SETTINGS` in shared (Task 2 table)
  - `func LoadSettings(path string) Settings` — missing file → defaults; corrupt file → rename to `<path>.bak`, return defaults
  - `func (s Settings) Save(path string) error` — atomic (tmp + rename), creates parent dirs

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/engine/settings_test.go`:
```go
package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsMatchSpec(t *testing.T) {
	s := DefaultSettings()
	if !s.General.LaunchAtLogin || !s.General.ShowMenuBarIcon || !s.General.EnableAnimations {
		t.Fatal("general defaults on")
	}
	if s.General.WindowPadding != 0 || s.General.PadFullscreen {
		t.Fatal("padding defaults")
	}
	if s.Snap.ZoneThickness != 10 || s.Snap.ActivationDelayMs != 0 || !s.Snap.RestorePreviousSize {
		t.Fatal("snap defaults")
	}
	if s.Snap.Zones[ZoneLeft] != ActionHalfLeft || s.Snap.Zones[ZoneTop] != ActionFullscreen || s.Snap.Zones[ZoneBottom] != ActionNone {
		t.Fatal("zone defaults")
	}
	if s.Hotkeys.Bindings[ActionCenter] != (Hotkey{KeyCode: 8, Modifiers: 256 + 2048}) {
		t.Fatal("center hotkey ⌥⌘C")
	}
	if len(s.Hotkeys.Bindings) != 17 {
		t.Fatalf("17 default bindings, got %d", len(s.Hotkeys.Bindings))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	s := DefaultSettings()
	s.General.WindowPadding = 12
	s.Snap.Zones[ZoneBottom] = ActionHalfBottom
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings(path)
	if got.General.WindowPadding != 12 || got.Snap.Zones[ZoneBottom] != ActionHalfBottom {
		t.Fatalf("round trip lost data: %+v", got)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	got := LoadSettings(filepath.Join(t.TempDir(), "nope.json"))
	if got.Snap.ZoneThickness != 10 {
		t.Fatal("want defaults")
	}
}

func TestLoadCorruptBacksUpAndDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	got := LoadSettings(path)
	if got.Snap.ZoneThickness != 10 {
		t.Fatal("want defaults on corrupt file")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("corrupt file should be backed up")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build.

- [ ] **Step 3: Implement**

`apps/desktop/internal/engine/settings.go`:
```go
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
	modShif = 512
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -v`
Expected: all engine tests PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(engine): settings model with atomic persistence and corrupt-file recovery"
```

---

### Task 8: Engine — snap-zone hit-testing

**Files:**
- Create: `apps/desktop/internal/engine/snapzones.go`, `apps/desktop/internal/engine/snapzones_test.go`

**Interfaces:**
- Consumes: `Point`, `Rect`, `Display`, `Zone`, `Action`, `SnapSettings` (Tasks 3, 7)
- Produces:
  - `func ZoneAt(p Point, d Display, thickness float64) (Zone, bool)` — which active-edge zone the cursor is in. Rule: a point within `thickness` of the top/bottom edge belongs to that edge's row; the row splits horizontally 25% / 50% / 25% into corner / edge / corner. Otherwise a point within `thickness` of the left/right edge splits vertically 25% / 50% / 25% the same way. Uses `d.Frame` (screen edges, not visible frame).
  - `func SnapAction(zone Zone, s SnapSettings, modThirds bool) Action` — maps zone → configured action; when `modThirds` is true (⌥ or ⌘ held) the plain `left`/`right` edge zones return `ActionNone`-safe third placement markers `ActionTwoThirdsLeft`? **No** — it returns the special values `ActionSnapThirdLeft`/`ActionSnapThirdRight` defined here as `Action` constants `"snap-third-left"` / `"snap-third-right"` (resolved to `ThirdColumn(0|2)` frames by the controller in Task 9). Disabled zones return `ActionNone`.

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/engine/snapzones_test.go`:
```go
package engine

import "testing"

func zone(t *testing.T, x, y float64) Zone {
	t.Helper()
	z, ok := ZoneAt(Point{x, y}, disp, 10)
	if !ok {
		t.Fatalf("expected a zone at %v,%v", x, y)
	}
	return z
}

func TestZoneRows(t *testing.T) {
	// Top edge row (y within 10 of top): 25/50/25 split of width 1440 → 360/720/360.
	if zone(t, 100, 5) != ZoneTopLeft || zone(t, 720, 5) != ZoneTop || zone(t, 1400, 5) != ZoneTopRight {
		t.Fatal("top row")
	}
	if zone(t, 100, 897) != ZoneBottomLeft || zone(t, 720, 897) != ZoneBottom || zone(t, 1400, 897) != ZoneBottomRight {
		t.Fatal("bottom row")
	}
	// Side edges: 25/50/25 split of height 900 → 225/450/225.
	if zone(t, 5, 100) != ZoneTopLeft || zone(t, 5, 450) != ZoneLeft || zone(t, 5, 800) != ZoneBottomLeft {
		t.Fatal("left edge")
	}
	if zone(t, 1435, 450) != ZoneRight {
		t.Fatal("right edge")
	}
}

func TestZoneMissesInterior(t *testing.T) {
	if _, ok := ZoneAt(Point{720, 450}, disp, 10); ok {
		t.Fatal("screen middle is no zone")
	}
	if _, ok := ZoneAt(Point{720, 12}, disp, 10); ok {
		t.Fatal("just past thickness is no zone")
	}
}

func TestSnapActionMapping(t *testing.T) {
	s := DefaultSettings().Snap
	if SnapAction(ZoneLeft, s, false) != ActionHalfLeft {
		t.Fatal("left → half-left")
	}
	if SnapAction(ZoneBottom, s, false) != ActionNone {
		t.Fatal("disabled zone → none")
	}
	if SnapAction(ZoneLeft, s, true) != ActionSnapThirdLeft || SnapAction(ZoneRight, s, true) != ActionSnapThirdRight {
		t.Fatal("modifier → thirds on side edges")
	}
	if SnapAction(ZoneTopLeft, s, true) != ActionUpperLeft {
		t.Fatal("modifier does not affect corners")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build.

- [ ] **Step 3: Implement**

`apps/desktop/internal/engine/snapzones.go`:
```go
package engine

// Special snap-only pseudo-actions produced when ⌥/⌘ is held while dragging
// to a plain side edge; the snap controller resolves them to third columns.
const (
	ActionSnapThirdLeft  Action = "snap-third-left"
	ActionSnapThirdRight Action = "snap-third-right"
)

// ZoneAt reports which active-edge zone contains p on display d. Corner zones
// take the outer 25% of each edge, the plain edge zone the middle 50%.
func ZoneAt(p Point, d Display, thickness float64) (Zone, bool) {
	f := d.Frame
	if !f.Contains(p) {
		return "", false
	}
	hx := (p.X - f.X) / f.W
	hy := (p.Y - f.Y) / f.H
	switch {
	case p.Y <= f.Y+thickness: // top row
		switch {
		case hx < 0.25:
			return ZoneTopLeft, true
		case hx > 0.75:
			return ZoneTopRight, true
		default:
			return ZoneTop, true
		}
	case p.Y >= f.Y+f.H-thickness: // bottom row
		switch {
		case hx < 0.25:
			return ZoneBottomLeft, true
		case hx > 0.75:
			return ZoneBottomRight, true
		default:
			return ZoneBottom, true
		}
	case p.X <= f.X+thickness: // left edge
		switch {
		case hy < 0.25:
			return ZoneTopLeft, true
		case hy > 0.75:
			return ZoneBottomLeft, true
		default:
			return ZoneLeft, true
		}
	case p.X >= f.X+f.W-thickness: // right edge
		switch {
		case hy < 0.25:
			return ZoneTopRight, true
		case hy > 0.75:
			return ZoneBottomRight, true
		default:
			return ZoneRight, true
		}
	}
	return "", false
}

// SnapAction maps a zone to its configured action. With ⌥/⌘ held (modThirds),
// plain left/right edges snap to thirds instead of their configured action.
func SnapAction(zone Zone, s SnapSettings, modThirds bool) Action {
	if modThirds {
		switch zone {
		case ZoneLeft:
			return ActionSnapThirdLeft
		case ZoneRight:
			return ActionSnapThirdRight
		}
	}
	a, ok := s.Zones[zone]
	if !ok {
		return ActionNone
	}
	return a
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -run 'Zone|SnapAction' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(engine): snap-zone hit-testing with modifier thirds"
```

---

### Task 9: Engine — snap controller state machine (delay, overlay, restore-previous-size)

**Files:**
- Create: `apps/desktop/internal/engine/snapctl.go`, `apps/desktop/internal/engine/snapctl_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces:
  - `type SnapPlatform interface { ShowOverlay(frame Rect); HideOverlay(); Now() int64 }` (Now returns milliseconds; injected for testability)
  - `type DragWindow struct { ID uint32; Frame Rect }`
  - `type SnapController struct` with `NewSnapController(p SnapPlatform, getSettings func() Settings, displays func() []Display)` and methods:
    - `DragStart(w DragWindow)` — remembers the window; if `restorePreviousSize` is on and this window was previously snapped by us and still has its snapped frame, returns `(Rect, bool)` restore frame (original size, positioned under the cursor is done by the caller in Task 15 — here just original size marker): signature `DragStart(w DragWindow) (restoreSize Rect, restore bool)`
    - `DragMove(cursor Point, modThirds bool)` — zone tracking; shows/hides overlay after `activationDelayMs` (re-arms when the zone changes)
    - `DragEnd(cursor Point, modThirds bool) (winID uint32, frame Rect, apply bool)` — if released in an armed zone, returns the frame to apply and records it for restore-previous-size
  - The controller is a pure state machine — **no timers**; delay is evaluated against `p.Now()` on each `DragMove` call.

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/engine/snapctl_test.go`:
```go
package engine

import "testing"

type fakePlat struct {
	now      int64
	shown    []Rect
	hidden   int
}

func (f *fakePlat) ShowOverlay(r Rect) { f.shown = append(f.shown, r) }
func (f *fakePlat) HideOverlay()       { f.hidden++ }
func (f *fakePlat) Now() int64         { return f.now }

func newCtl(f *fakePlat, mut func(*Settings)) *SnapController {
	s := DefaultSettings()
	if mut != nil {
		mut(&s)
	}
	return NewSnapController(f, func() Settings { return s }, func() []Display { return []Display{disp} })
}

func TestSnapLeftEdgeShowsOverlayAndApplies(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 7, Frame: Rect{100, 100, 400, 300}})
	c.DragMove(Point{5, 450}, false)
	if len(f.shown) != 1 {
		t.Fatal("overlay shown on zone enter")
	}
	half, _ := FrameFor(ActionHalfLeft, Rect{}, disp, 0, false)
	eq(t, f.shown[0], half)
	win, frame, apply := c.DragEnd(Point{5, 450}, false)
	if !apply || win != 7 {
		t.Fatal("apply on drop in zone")
	}
	eq(t, frame, half)
	if f.hidden == 0 {
		t.Fatal("overlay hidden after drop")
	}
}

func TestActivationDelayArmsLate(t *testing.T) {
	f := &fakePlat{now: 1000}
	c := newCtl(f, func(s *Settings) { s.Snap.ActivationDelayMs = 200 })
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	if len(f.shown) != 0 {
		t.Fatal("not armed before delay")
	}
	if _, _, apply := c.DragEnd(Point{5, 450}, false); apply {
		t.Fatal("drop before delay does nothing")
	}
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	f.now = 1300
	c.DragMove(Point{6, 450}, false)
	if len(f.shown) == 0 {
		t.Fatal("armed after delay elapsed")
	}
}

func TestLeavingZoneHidesOverlay(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	c.DragMove(Point{700, 450}, false)
	if f.hidden == 0 {
		t.Fatal("overlay hides when leaving zone")
	}
	if _, _, apply := c.DragEnd(Point{700, 450}, false); apply {
		t.Fatal("no apply outside zone")
	}
}

func TestModifierThirdsOnSideEdge(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, true)
	_, frame, apply := c.DragEnd(Point{5, 450}, true)
	if !apply {
		t.Fatal("apply")
	}
	eq(t, frame, ThirdColumn(0, disp, 0))
}

func TestDisabledZoneDoesNothing(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{720, 897}, false) // bottom zone: default "none"
	if len(f.shown) != 0 {
		t.Fatal("disabled zone shows no overlay")
	}
}

func TestRestorePreviousSize(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	orig := Rect{100, 100, 400, 300}
	c.DragStart(DragWindow{ID: 7, Frame: orig})
	c.DragMove(Point{5, 450}, false)
	_, snapped, _ := c.DragEnd(Point{5, 450}, false)
	// Next drag of the same window while still snapped → restore original size.
	restore, ok := c.DragStart(DragWindow{ID: 7, Frame: snapped})
	if !ok || restore.W != orig.W || restore.H != orig.H {
		t.Fatalf("want restore to original size, got %+v %v", restore, ok)
	}
	// A window moved since snapping does not restore.
	if _, ok := c.DragStart(DragWindow{ID: 7, Frame: orig}); ok {
		t.Fatal("frame changed since snap → no restore")
	}
}

func TestSnapDisabledGloballyIgnoresEverything(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, func(s *Settings) { s.Snap.Enabled = false })
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	if len(f.shown) != 0 {
		t.Fatal("disabled snap shows nothing")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/engine/`
Expected: FAIL to build.

- [ ] **Step 3: Implement**

`apps/desktop/internal/engine/snapctl.go`:
```go
package engine

import "sync"

// SnapPlatform is what the controller needs from the native layer.
type SnapPlatform interface {
	ShowOverlay(frame Rect)
	HideOverlay()
	Now() int64 // milliseconds
}

type DragWindow struct {
	ID    uint32
	Frame Rect
}

type snapRecord struct{ snapped, original Rect }

// SnapController is a pure state machine driving snap-to-edges. The platform
// layer feeds it DragStart/DragMove/DragEnd from the event tap; it decides
// when to show the overlay and what frame to apply. No goroutines, no timers.
type SnapController struct {
	mu          sync.Mutex
	plat        SnapPlatform
	getSettings func() Settings
	displays    func() []Display

	dragging   bool
	win        DragWindow
	zone       Zone
	zoneSince  int64
	armed      bool
	armedFrame Rect
	history    map[uint32]snapRecord
}

func NewSnapController(p SnapPlatform, getSettings func() Settings, displays func() []Display) *SnapController {
	return &SnapController{plat: p, getSettings: getSettings, displays: displays, history: map[uint32]snapRecord{}}
}

func (c *SnapController) DragStart(w DragWindow) (Rect, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.getSettings()
	if !s.Snap.Enabled {
		return Rect{}, false
	}
	c.dragging, c.win, c.zone, c.armed = true, w, "", false
	if rec, ok := c.history[w.ID]; ok {
		delete(c.history, w.ID)
		if s.Snap.RestorePreviousSize && w.Frame.Eq(rec.snapped, 2.0) {
			return Rect{W: rec.original.W, H: rec.original.H}, true
		}
	}
	return Rect{}, false
}

// resolve computes the target frame for the zone under the cursor, or ok=false.
func (c *SnapController) resolve(cursor Point, modThirds bool) (Rect, bool) {
	s := c.getSettings()
	for _, d := range c.displays() {
		zone, ok := ZoneAt(cursor, d, s.Snap.ZoneThickness)
		if !ok {
			continue
		}
		if zone != c.zone {
			c.zone, c.zoneSince = zone, c.plat.Now()
		}
		if c.plat.Now()-c.zoneSince < int64(s.Snap.ActivationDelayMs) {
			return Rect{}, false
		}
		switch a := SnapAction(zone, s.Snap, modThirds); a {
		case ActionNone:
			return Rect{}, false
		case ActionSnapThirdLeft:
			return ThirdColumn(0, d, s.General.WindowPadding), true
		case ActionSnapThirdRight:
			return ThirdColumn(2, d, s.General.WindowPadding), true
		default:
			if f, ok := FrameFor(a, c.win.Frame, d, s.General.WindowPadding, s.General.PadFullscreen); ok {
				return f, true
			}
			return Rect{}, false
		}
	}
	c.zone = ""
	return Rect{}, false
}

func (c *SnapController) DragMove(cursor Point, modThirds bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dragging {
		return
	}
	frame, ok := c.resolve(cursor, modThirds)
	switch {
	case ok && (!c.armed || !frame.Eq(c.armedFrame, 0.5)):
		c.armed, c.armedFrame = true, frame
		c.plat.ShowOverlay(frame)
	case !ok && c.armed:
		c.armed = false
		c.plat.HideOverlay()
	}
}

func (c *SnapController) DragEnd(cursor Point, modThirds bool) (uint32, Rect, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dragging {
		return 0, Rect{}, false
	}
	c.dragging = false
	if c.armed {
		c.armed = false
		c.plat.HideOverlay()
	}
	frame, ok := c.resolve(cursor, modThirds)
	if !ok {
		return 0, Rect{}, false
	}
	c.history[c.win.ID] = snapRecord{snapped: frame, original: c.win.Frame}
	return c.win.ID, frame, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/engine/ -v`
Expected: all engine tests PASS (including all 8 snap-controller tests).

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(engine): snap controller state machine with delay, overlay, restore"
```

---

### Task 10: Platform — AX window access + displays + probe CLI

**Files:**
- Create: `apps/desktop/internal/platform/ax_darwin.go`, `apps/desktop/internal/platform/ax_darwin.h`, `apps/desktop/internal/platform/displays_darwin.go`, `apps/desktop/internal/platform/displays_darwin.h`, `apps/desktop/cmd/axprobe/main.go`

**Interfaces:**
- Consumes: `engine.Rect`, `engine.Display`, `engine.Point`
- Produces:
  - `func AXTrusted(prompt bool) bool`
  - `type Window struct { ID uint32; ref unsafe pointer (unexported) }` with `func (w *Window) Frame() (engine.Rect, error)`, `func (w *Window) SetFrame(r engine.Rect) error`, `func (w *Window) Release()`
  - `func FocusedWindow() (*Window, error)` — frontmost app's focused AX window
  - `func WindowAt(p engine.Point) (*Window, error)` — AX window under a screen point
  - `func Displays() []engine.Display` — all screens in **top-left-origin** coordinates

AX already uses top-left-origin global coordinates, so `Frame`/`SetFrame` need no conversion; only `Displays()` converts from Cocoa's bottom-left origin (`topLeftY = primaryHeight - (cocoaY + height)`).

- [ ] **Step 1: Write the headers**

`apps/desktop/internal/platform/ax_darwin.h`:
```objc
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

static void ts_release(AXUIElementRef ref) { if (ref) CFRelease(ref); }
```

`apps/desktop/internal/platform/displays_darwin.h`:
```objc
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
```

- [ ] **Step 2: Write the Go wrappers**

`apps/desktop/internal/platform/ax_darwin.go`:
```go
//go:build darwin

// Package platform is the cgo/Objective-C bridge to macOS. It contains no
// logic beyond translation — all decisions live in internal/engine.
package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework Carbon
#include "ax_darwin.h"
*/
import "C"

import (
	"errors"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var ErrNoWindow = errors.New("platform: no accessible window")

// AXTrusted reports (and with prompt=true requests) Accessibility permission.
func AXTrusted(prompt bool) bool { return bool(C.ts_ax_trusted(C.bool(prompt))) }

type Window struct {
	ID  uint32
	ref C.AXUIElementRef
}

func wrap(ref C.AXUIElementRef) (*Window, error) {
	if ref == 0 {
		return nil, ErrNoWindow
	}
	return &Window{ID: uint32(C.ts_window_id(ref)), ref: ref}, nil
}

func FocusedWindow() (*Window, error) { return wrap(C.ts_focused_window()) }

func WindowAt(p engine.Point) (*Window, error) {
	return wrap(C.ts_window_at(C.double(p.X), C.double(p.Y)))
}

func (w *Window) Frame() (engine.Rect, error) {
	f := C.ts_window_frame(w.ref)
	if f.ok == 0 {
		return engine.Rect{}, ErrNoWindow
	}
	return engine.Rect{X: float64(f.x), Y: float64(f.y), W: float64(f.w), H: float64(f.h)}, nil
}

func (w *Window) SetFrame(r engine.Rect) error {
	if !bool(C.ts_set_window_frame(w.ref, C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H))) {
		return errors.New("platform: window rejected frame (not resizable?)")
	}
	return nil
}

func (w *Window) Release() { C.ts_release(w.ref); w.ref = 0 }
```

`apps/desktop/internal/platform/displays_darwin.go`:
```go
//go:build darwin

package platform

/*
#include "displays_darwin.h"
*/
import "C"

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// Displays returns all screens in top-left-origin coordinates.
func Displays() []engine.Display {
	var buf [16]C.TSDisplay
	n := int(C.ts_displays(&buf[0], 16))
	out := make([]engine.Display, 0, n)
	for i := 0; i < n; i++ {
		d := buf[i]
		out = append(out, engine.Display{
			ID:      uint32(d.id),
			Frame:   engine.Rect{X: float64(d.fx), Y: float64(d.fy), W: float64(d.fw), H: float64(d.fh)},
			Visible: engine.Rect{X: float64(d.vx), Y: float64(d.vy), W: float64(d.vw), H: float64(d.vh)},
		})
	}
	return out
}
```

- [ ] **Step 3: Write the probe CLI (manual verification harness for all platform tasks)**

`apps/desktop/cmd/axprobe/main.go`:
```go
// axprobe is a manual smoke-test harness for the cgo platform layer.
// Usage: go run ./cmd/axprobe [displays|focused|snapleft]
package main

import (
	"fmt"
	"os"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

func main() {
	cmd := "displays"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	fmt.Println("AX trusted:", platform.AXTrusted(true))
	switch cmd {
	case "displays":
		for i, d := range platform.Displays() {
			fmt.Printf("display %d: id=%d frame=%+v visible=%+v\n", i, d.ID, d.Frame, d.Visible)
		}
	case "focused":
		w, err := platform.FocusedWindow()
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer w.Release()
		f, _ := w.Frame()
		fmt.Printf("focused window id=%d frame=%+v\n", w.ID, f)
	case "snapleft":
		w, err := platform.FocusedWindow()
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer w.Release()
		f, _ := w.Frame()
		d := platform.Displays()[engine.DisplayOf(f, platform.Displays())]
		target, _ := engine.FrameFor(engine.ActionHalfLeft, f, d, 0, false)
		fmt.Println("setting frame:", target, "err:", w.SetFrame(target))
	}
}
```

- [ ] **Step 4: Build and manually verify**

Run: `cd apps/desktop && go vet ./... && go build ./... && go run ./cmd/axprobe displays`
Expected: builds clean; prints your display(s) with sensible top-left-origin frames (primary starts at 0,0; visible Y > 0 because of the menu bar).

Run: `go run ./cmd/axprobe snapleft` from a terminal that has Accessibility permission (grant it when prompted, then rerun).
Expected: the frontmost window (e.g. the terminal itself) snaps to the left half of its screen.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(platform): AX window access, display enumeration, axprobe CLI"
```

---

### Task 11: Platform — Carbon global hotkeys

**Files:**
- Create: `apps/desktop/internal/platform/hotkeys_darwin.go`, `apps/desktop/internal/platform/hotkeys_darwin.h`

**Interfaces:**
- Consumes: `engine.Hotkey`
- Produces:
  - `func InstallHotkeyHandler(fire func(id uint32))` — call once at startup, before any Register
  - `func RegisterHotkey(id uint32, hk engine.Hotkey) error` — id is caller-chosen (Task 15 uses the action's index)
  - `func UnregisterAllHotkeys()`

- [ ] **Step 1: Write the header**

`apps/desktop/internal/platform/hotkeys_darwin.h`:
```objc
#import <Carbon/Carbon.h>

extern void goHotkeyFired(unsigned int id); // exported from Go

static OSStatus ts_hk_handler(EventHandlerCallRef next, EventRef evt, void *data) {
  EventHotKeyID hk;
  GetEventParameter(evt, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(hk), NULL, &hk);
  goHotkeyFired(hk.id);
  return noErr;
}

static void ts_hk_install(void) {
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
```

- [ ] **Step 2: Write the Go wrapper**

`apps/desktop/internal/platform/hotkeys_darwin.go`:
```go
//go:build darwin

package platform

/*
#include "hotkeys_darwin.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var (
	hotkeyMu   sync.Mutex
	hotkeyRefs []unsafe.Pointer
	hotkeyFire func(id uint32)
)

//export goHotkeyFired
func goHotkeyFired(id C.uint) {
	hotkeyMu.Lock()
	fire := hotkeyFire
	hotkeyMu.Unlock()
	if fire != nil {
		go fire(uint32(id))
	}
}

// InstallHotkeyHandler installs the Carbon event handler. Call once, from the
// main thread, before any RegisterHotkey.
func InstallHotkeyHandler(fire func(id uint32)) {
	hotkeyMu.Lock()
	hotkeyFire = fire
	hotkeyMu.Unlock()
	C.ts_hk_install()
}

func RegisterHotkey(id uint32, hk engine.Hotkey) error {
	ref := C.ts_hk_register(C.uint(id), C.uint(hk.KeyCode), C.uint(hk.Modifiers))
	if ref == nil {
		return fmt.Errorf("platform: hotkey %d (key %d mods %d) rejected — already taken?", id, hk.KeyCode, hk.Modifiers)
	}
	hotkeyMu.Lock()
	hotkeyRefs = append(hotkeyRefs, ref)
	hotkeyMu.Unlock()
	return nil
}

func UnregisterAllHotkeys() {
	hotkeyMu.Lock()
	defer hotkeyMu.Unlock()
	for _, ref := range hotkeyRefs {
		C.ts_hk_unregister(ref)
	}
	hotkeyRefs = nil
}
```

Note: cgo requires the `//export` comment directly above the function with no blank line, and `hotkeys_darwin.h` must NOT be included from a file that also defines exported functions' prototypes differently — the pattern above (extern declaration in the .h) is the standard one.

- [ ] **Step 3: Verify with axprobe**

Add a `hotkeys` case to `apps/desktop/cmd/axprobe/main.go` (before the final `}` of the switch):
```go
	case "hotkeys":
		platform.InstallHotkeyHandler(func(id uint32) { fmt.Println("hotkey fired:", id) })
		if err := platform.RegisterHotkey(1, engine.Hotkey{KeyCode: 8, Modifiers: 2048 + 256}); err != nil { // ⌥⌘C
			fmt.Println("register:", err)
			return
		}
		fmt.Println("press ⌥⌘C (ctrl-c to quit)")
		select {}
```
Run: `go run ./cmd/axprobe hotkeys`, press ⌥⌘C.
Expected: prints `hotkey fired: 1`. (Note: Carbon hotkeys need a run loop — if nothing fires, wrap the `select{}` with a `C` run loop instead: this is resolved for real in Task 15 where Wails runs the main run loop; for the probe, add `#include <CoreFoundation/CoreFoundation.h>` usage via `C.CFRunLoopRun()` exposed as `platform.RunLoop()` helper in `hotkeys_darwin.go`:
```go
// RunLoop blocks running the current thread's CFRunLoop (probe/testing only).
func RunLoop() { C.CFRunLoopRun() }
```
and call `platform.RunLoop()` instead of `select {}`.)

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat(platform): Carbon global hotkey registration"
```

---

### Task 12: Platform — CGEventTap drag watcher

**Files:**
- Create: `apps/desktop/internal/platform/drag_darwin.go`, `apps/desktop/internal/platform/drag_darwin.h`

**Interfaces:**
- Consumes: nothing from engine (raw events only — the app layer connects it to `SnapController`)
- Produces:
  - `type DragEventKind int` with `DragDown`, `DragMoved`, `DragUp` constants
  - `func StartDragTap(handler func(kind DragEventKind, x, y float64, modThirds bool)) error` — listen-only session event tap on the main run loop; auto-re-enables after `kCGEventTapDisabledByTimeout`; `modThirds` is true when ⌥ or ⌘ is in the event flags
  - `func StopDragTap()`

- [ ] **Step 1: Write the header**

`apps/desktop/internal/platform/drag_darwin.h`:
```objc
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
```

- [ ] **Step 2: Write the Go wrapper**

`apps/desktop/internal/platform/drag_darwin.go`:
```go
//go:build darwin

package platform

/*
#include "drag_darwin.h"
*/
import "C"

import (
	"errors"
	"sync"
)

type DragEventKind int

const (
	DragDown DragEventKind = iota
	DragMoved
	DragUp
)

var (
	dragMu      sync.Mutex
	dragHandler func(kind DragEventKind, x, y float64, modThirds bool)
)

//export goDragEvent
func goDragEvent(kind C.int, x, y C.double, modThirds C.int) {
	dragMu.Lock()
	h := dragHandler
	dragMu.Unlock()
	if h != nil {
		h(DragEventKind(kind), float64(x), float64(y), modThirds != 0)
	}
}

// StartDragTap installs a listen-only session event tap on the main run loop.
// Requires Accessibility permission. The handler runs on the tap thread — keep
// it fast (the SnapController is lock-cheap and does no AX calls on DragMoved).
func StartDragTap(handler func(kind DragEventKind, x, y float64, modThirds bool)) error {
	dragMu.Lock()
	dragHandler = handler
	dragMu.Unlock()
	if !bool(C.ts_tap_start()) {
		return errors.New("platform: CGEventTapCreate failed (missing Accessibility permission?)")
	}
	return nil
}

func StopDragTap() {
	C.ts_tap_stop()
	dragMu.Lock()
	dragHandler = nil
	dragMu.Unlock()
}
```

- [ ] **Step 3: Verify with axprobe**

Add a `drag` case to axprobe's switch:
```go
	case "drag":
		err := platform.StartDragTap(func(kind platform.DragEventKind, x, y float64, mod bool) {
			fmt.Printf("drag kind=%d x=%.0f y=%.0f mod=%v\n", kind, x, y, mod)
		})
		fmt.Println("tap started, err:", err, "— drag any window (ctrl-c to quit)")
		platform.RunLoop()
```
Run: `go run ./cmd/axprobe drag`, click and drag anywhere; hold ⌥ while dragging.
Expected: down/moved/up events stream with correct coordinates; `mod=true` while ⌥ held.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat(platform): listen-only CGEventTap drag watcher with auto re-enable"
```

---

### Task 13: Platform — snap-preview overlay window

**Files:**
- Create: `apps/desktop/internal/platform/overlay_darwin.go`, `apps/desktop/internal/platform/overlay_darwin.h`

**Interfaces:**
- Consumes: `engine.Rect`
- Produces: `func ShowOverlay(r engine.Rect)`, `func HideOverlay()` — thread-safe (dispatch to main queue); translucent rounded-rect borderless panel that ignores mouse events and floats above normal windows. Satisfies `engine.SnapPlatform` (with a `Now()` added at the call site in Task 15).

- [ ] **Step 1: Write the header**

`apps/desktop/internal/platform/overlay_darwin.h`:
```objc
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
```

- [ ] **Step 2: Write the Go wrapper**

`apps/desktop/internal/platform/overlay_darwin.go`:
```go
//go:build darwin

package platform

/*
#include "overlay_darwin.h"
*/
import "C"

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// ShowOverlay shows (or moves) the translucent snap preview. Safe from any thread.
func ShowOverlay(r engine.Rect) {
	C.ts_overlay_show(C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H))
}

func HideOverlay() { C.ts_overlay_hide() }
```

- [ ] **Step 3: Verify with axprobe**

Add an `overlay` case:
```go
	case "overlay":
		d := platform.Displays()[0]
		half, _ := engine.FrameFor(engine.ActionHalfLeft, engine.Rect{}, d, 0, false)
		platform.ShowOverlay(half)
		fmt.Println("overlay on left half for 3s")
		go func() { time.Sleep(3 * time.Second); platform.HideOverlay(); time.Sleep(time.Second); os.Exit(0) }()
		platform.RunLoop()
```
(add `"time"` to imports)
Run: `go run ./cmd/axprobe overlay`
Expected: translucent rounded rectangle covers the left half of the screen for 3 seconds. NOTE: NSPanel needs an NSApplication — if nothing appears, add `static void ts_app_init(void) { [NSApplication sharedApplication]; }` to `overlay_darwin.h`, expose `platform.AppInit()`, and call it first in axprobe (the real app has NSApp via Wails).

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat(platform): translucent snap-preview overlay panel"
```

---

### Task 14: Platform — login item, activation policy, animated frame setter

**Files:**
- Create: `apps/desktop/internal/platform/system_darwin.go`, `apps/desktop/internal/platform/system_darwin.h`, `apps/desktop/internal/platform/animate.go`, `apps/desktop/internal/platform/animate_test.go`

**Interfaces:**
- Consumes: `Window`, `engine.Rect`
- Produces:
  - `func SetLoginItem(enabled bool) error` — `SMAppService.mainApp` register/unregister (macOS 13+)
  - `func ActivatePrefs()` — activation policy Regular + activate (so the prefs window can take focus)
  - `func HideFromDock()` — activation policy Accessory
  - `func AnimateFrame(w *Window, from, to engine.Rect, enabled bool, sleep func(ms int))` — 8-step ease-out interpolation over ~150 ms when enabled, single SetFrame when not. `sleep` injected for the unit test.

- [ ] **Step 1: Write the header**

`apps/desktop/internal/platform/system_darwin.h`:
```objc
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
```

- [ ] **Step 2: Write the Go wrappers**

`apps/desktop/internal/platform/system_darwin.go`:
```go
//go:build darwin

package platform

/*
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
func HideFromDock() { C.ts_hide_from_dock() }
```

`apps/desktop/internal/platform/animate.go` (pure Go — testable):
```go
package platform

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// FrameSetter abstracts *Window for testing.
type FrameSetter interface{ SetFrame(engine.Rect) error }

const animSteps = 8

// AnimateFrame moves a window from → to. With animation enabled it eases out
// over animSteps steps (~150ms via sleep(18)); otherwise one direct set.
func AnimateFrame(w FrameSetter, from, to engine.Rect, enabled bool, sleep func(ms int)) {
	if !enabled {
		_ = w.SetFrame(to)
		return
	}
	for i := 1; i <= animSteps; i++ {
		t := float64(i) / animSteps
		t = 1 - (1-t)*(1-t) // ease-out
		_ = w.SetFrame(engine.Rect{
			X: from.X + (to.X-from.X)*t,
			Y: from.Y + (to.Y-from.Y)*t,
			W: from.W + (to.W-from.W)*t,
			H: from.H + (to.H-from.H)*t,
		})
		if i < animSteps {
			sleep(18)
		}
	}
}
```

- [ ] **Step 3: Write and run the animation unit test**

`apps/desktop/internal/platform/animate_test.go`:
```go
package platform

import (
	"testing"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

type recorder struct{ frames []engine.Rect }

func (r *recorder) SetFrame(f engine.Rect) error { r.frames = append(r.frames, f); return nil }

func TestAnimateDisabledSetsOnce(t *testing.T) {
	r := &recorder{}
	AnimateFrame(r, engine.Rect{}, engine.Rect{X: 100, W: 50, H: 50}, false, func(int) {})
	if len(r.frames) != 1 || r.frames[0].X != 100 {
		t.Fatalf("want single direct set, got %+v", r.frames)
	}
}

func TestAnimateEnabledEndsExactlyAtTarget(t *testing.T) {
	r := &recorder{}
	to := engine.Rect{X: 100, Y: 200, W: 300, H: 400}
	AnimateFrame(r, engine.Rect{}, to, true, func(int) {})
	if len(r.frames) != animSteps {
		t.Fatalf("want %d steps, got %d", animSteps, len(r.frames))
	}
	if !r.frames[len(r.frames)-1].Eq(to, 0.001) {
		t.Fatalf("final frame must equal target, got %+v", r.frames[len(r.frames)-1])
	}
	if r.frames[0].X <= 0 {
		t.Fatal("first step must move toward target")
	}
}
```

Run: `cd apps/desktop && go test ./internal/platform/ -run Animate -v`
Expected: PASS (this file is pure Go; cgo files still compile on darwin).

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat(platform): login item, activation policy, animated frame setter"
```

---

### Task 15: App — action dispatcher (glue engine ⇄ platform)

**Files:**
- Create: `apps/desktop/internal/app/dispatcher.go`, `apps/desktop/internal/app/dispatcher_test.go`, `apps/desktop/internal/app/animate_shim.go`

**Interfaces:**
- Consumes: `engine.*` (Tasks 3–9); `platform` functions are injected via an interface so the dispatcher is unit-testable.
- Produces:
  - `type Platform interface { FocusedWindow() (AppWindow, error); WindowAt(engine.Point) (AppWindow, error); Displays() []engine.Display; ShowOverlay(engine.Rect); HideOverlay() }`
  - `type AppWindow interface { WinID() uint32; Frame() (engine.Rect, error); SetFrame(engine.Rect) error; Release() }`
  - `type Dispatcher struct` with `NewDispatcher(p Platform, getSettings func() engine.Settings)` and:
    - `Perform(a engine.Action)` — executes any of the 17 actions on the focused window (undo push included)
    - `Undo()`
    - `Snap() *engine.SnapController` accessor, and methods `OnDragEvent(kind int, x, y float64, modThirds bool)` wiring tap events → SnapController → AX apply (drag-start resolves the window under the cursor once, on mouse-down)
    - animation is applied through `platform.AnimateFrame` with `time.Sleep`; injected as `animate func(w AppWindow, from, to engine.Rect)` for tests

- [ ] **Step 1: Write failing tests**

`apps/desktop/internal/app/dispatcher_test.go`:
```go
package app

import (
	"testing"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var disp = engine.Display{ID: 1, Frame: engine.Rect{0, 0, 1440, 900}, Visible: engine.Rect{0, 25, 1440, 815}}

type fakeWin struct {
	id    uint32
	frame engine.Rect
	sets  []engine.Rect
}

func (w *fakeWin) WinID() uint32                    { return w.id }
func (w *fakeWin) Frame() (engine.Rect, error)     { return w.frame, nil }
func (w *fakeWin) SetFrame(r engine.Rect) error    { w.frame = r; w.sets = append(w.sets, r); return nil }
func (w *fakeWin) Release()                        {}

type fakePlatform struct {
	win      *fakeWin
	overlays int
}

func (p *fakePlatform) FocusedWindow() (AppWindow, error)            { return p.win, nil }
func (p *fakePlatform) WindowAt(engine.Point) (AppWindow, error)     { return p.win, nil }
func (p *fakePlatform) Displays() []engine.Display                   { return []engine.Display{disp} }
func (p *fakePlatform) ShowOverlay(engine.Rect)                      { p.overlays++ }
func (p *fakePlatform) HideOverlay()                                 {}

func newTestDispatcher(w *fakeWin) (*Dispatcher, *fakePlatform) {
	p := &fakePlatform{win: w}
	s := engine.DefaultSettings()
	s.General.EnableAnimations = false
	d := NewDispatcher(p, func() engine.Settings { return s })
	return d, p
}

func TestPerformHalfLeftMovesWindow(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{100, 100, 400, 300}}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionHalfLeft)
	want, _ := engine.FrameFor(engine.ActionHalfLeft, engine.Rect{}, disp, 0, false)
	if !w.frame.Eq(want, 0.01) {
		t.Fatalf("got %+v want %+v", w.frame, want)
	}
}

func TestPerformThenUndoRestores(t *testing.T) {
	orig := engine.Rect{100, 100, 400, 300}
	w := &fakeWin{id: 1, frame: orig}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionFullscreen)
	d.Undo()
	if !w.frame.Eq(orig, 0.01) {
		t.Fatalf("undo should restore %+v, got %+v", orig, w.frame)
	}
}

func TestNextThirdCyclesViaDispatcher(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{100, 100, 400, 300}}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionNextThird)
	if !w.frame.Eq(engine.ThirdColumn(0, disp, 0), 0.01) {
		t.Fatal("first next-third → left column")
	}
	d.Perform(engine.ActionNextThird)
	if !w.frame.Eq(engine.ThirdColumn(1, disp, 0), 0.01) {
		t.Fatal("second next-third → middle column")
	}
}

func TestDragSnapEndToEnd(t *testing.T) {
	w := &fakeWin{id: 9, frame: engine.Rect{300, 300, 500, 400}}
	d, p := newTestDispatcher(w)
	d.OnDragEvent(0, 320, 310, false) // mouse-down on the window
	d.OnDragEvent(1, 5, 450, false)   // drag into left edge zone
	if p.overlays == 0 {
		t.Fatal("overlay shown while hovering zone")
	}
	d.OnDragEvent(2, 5, 450, false) // drop
	want, _ := engine.FrameFor(engine.ActionHalfLeft, engine.Rect{}, disp, 0, false)
	if !w.frame.Eq(want, 0.01) {
		t.Fatalf("dropped window snapped, got %+v", w.frame)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/desktop && go test ./internal/app/`
Expected: FAIL to build.

- [ ] **Step 3: Implement**

`apps/desktop/internal/app/dispatcher.go`:
```go
// Package app glues the pure engine to the native platform layer.
package app

import (
	"log/slog"
	"time"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

type AppWindow interface {
	WinID() uint32
	Frame() (engine.Rect, error)
	SetFrame(engine.Rect) error
	Release()
}

type Platform interface {
	FocusedWindow() (AppWindow, error)
	WindowAt(engine.Point) (AppWindow, error)
	Displays() []engine.Display
	ShowOverlay(engine.Rect)
	HideOverlay()
}

type Dispatcher struct {
	plat        Platform
	getSettings func() engine.Settings
	undo        *engine.UndoStack
	snap        *engine.SnapController
	animate     func(w AppWindow, from, to engine.Rect)

	dragWin AppWindow // window picked up at mouse-down, nil otherwise
}

// snapPlatformAdapter adds Now() to the overlay pair for the SnapController.
type snapPlatformAdapter struct{ p Platform }

func (a snapPlatformAdapter) ShowOverlay(r engine.Rect) { a.p.ShowOverlay(r) }
func (a snapPlatformAdapter) HideOverlay()              { a.p.HideOverlay() }
func (a snapPlatformAdapter) Now() int64                { return time.Now().UnixMilli() }

func NewDispatcher(p Platform, getSettings func() engine.Settings) *Dispatcher {
	d := &Dispatcher{
		plat:        p,
		getSettings: getSettings,
		undo:        engine.NewUndoStack(),
	}
	d.snap = engine.NewSnapController(snapPlatformAdapter{p}, getSettings, p.Displays)
	d.animate = func(w AppWindow, from, to engine.Rect) {
		enabled := getSettings().General.EnableAnimations
		animateFrame(w, from, to, enabled)
	}
	return d
}

// Perform runs one of the 17 actions on the currently focused window.
func (d *Dispatcher) Perform(a engine.Action) {
	w, err := d.plat.FocusedWindow()
	if err != nil {
		slog.Warn("no focused window", "action", a, "err", err)
		return
	}
	defer w.Release()
	cur, err := w.Frame()
	if err != nil {
		return
	}
	displays := d.plat.Displays()
	if len(displays) == 0 {
		return
	}
	s := d.getSettings()
	di := engine.DisplayOf(cur, displays)
	disp := displays[di]
	pad := s.General.WindowPadding

	var target engine.Rect
	var ok bool
	switch a {
	case engine.ActionNextThird, engine.ActionPrevThird:
		target, ok = engine.ThirdFrame(a, cur, disp, pad)
	case engine.ActionNextDisplay, engine.ActionPrevDisplay:
		to := engine.AdjacentDisplay(di, displays, a == engine.ActionNextDisplay)
		if to != di {
			target, ok = engine.MapToDisplay(cur, disp, displays[to]), true
		}
	default:
		target, ok = engine.FrameFor(a, cur, disp, pad, s.General.PadFullscreen)
	}
	if !ok {
		return
	}
	d.undo.Push(w.WinID(), cur)
	d.animate(w, cur, target)
}

func (d *Dispatcher) Undo() {
	w, err := d.plat.FocusedWindow()
	if err != nil {
		return
	}
	defer w.Release()
	prev, ok := d.undo.Pop(w.WinID())
	if !ok {
		return
	}
	cur, err := w.Frame()
	if err != nil {
		return
	}
	d.animate(w, cur, prev)
}

// OnDragEvent receives raw tap events (kind: 0=down 1=moved 2=up).
func (d *Dispatcher) OnDragEvent(kind int, x, y float64, modThirds bool) {
	p := engine.Point{X: x, Y: y}
	switch kind {
	case 0: // down: resolve the window once
		if d.dragWin != nil {
			d.dragWin.Release()
			d.dragWin = nil
		}
		w, err := d.plat.WindowAt(p)
		if err != nil {
			return
		}
		frame, err := w.Frame()
		if err != nil {
			w.Release()
			return
		}
		d.dragWin = w
		if restore, ok := d.snap.DragStart(engine.DragWindow{ID: w.WinID(), Frame: frame}); ok {
			// Restore previous size under the cursor (keep grab point proportional).
			_ = w.SetFrame(engine.Rect{X: x - restore.W/2, Y: frame.Y, W: restore.W, H: restore.H})
		}
	case 1:
		if d.dragWin != nil {
			d.snap.DragMove(p, modThirds)
		}
	case 2:
		if d.dragWin == nil {
			return
		}
		if _, frame, apply := d.snap.DragEnd(p, modThirds); apply {
			if cur, err := d.dragWin.Frame(); err == nil {
				d.undo.Push(d.dragWin.WinID(), cur)
				d.animate(d.dragWin, cur, frame)
			}
		}
		d.dragWin.Release()
		d.dragWin = nil
	}
}
```

Add `apps/desktop/internal/app/animate_shim.go` so the dispatcher stays testable off-macOS:
```go
package app

import (
	"time"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

func animateFrame(w AppWindow, from, to engine.Rect, enabled bool) {
	platform.AnimateFrame(w, from, to, enabled, func(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) })
}
```
(`platform.AnimateFrame` takes a `FrameSetter` interface — `AppWindow` satisfies it.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/desktop && go test ./internal/app/ -v`
Expected: all 4 dispatcher tests PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(app): action dispatcher wiring engine to platform with undo and drag snap"
```

---

### Task 16: App — Wails v3 bootstrap: systray, hotkeys, services, prefs window

**Files:**
- Create: `apps/desktop/main.go`, `apps/desktop/internal/app/settings_service.go`, `apps/desktop/internal/app/realplatform_darwin.go`, `apps/desktop/wails.json` (if `wails3` init requires it), placeholder `apps/desktop/frontend/dist/index.html`
- Modify: `apps/desktop/go.mod` (add wails v3)

**Interfaces:**
- Consumes: `Dispatcher` (Task 15), `platform.*` (Tasks 10–14), `engine.Settings`
- Produces:
  - `type SettingsService struct` (Wails-bound) with methods `Get() engine.Settings`, `Update(s engine.Settings) error` (persists + applies side effects + emits `settings:changed`), `RestoreDefaultHotkeys() engine.Settings`, `AXTrusted() bool`, `RequestAXPermission() bool`
  - a running menu-bar app: tray menu with all actions, working hotkeys, drag snapping, prefs window on ⌘, / second-launch
  - `applySideEffects(old, new engine.Settings)` — re-register hotkeys, login item, tray visibility

- [ ] **Step 1: Add Wails v3 and scaffold main.go**

Run: `cd apps/desktop && go get github.com/wailsapp/wails/v3@latest && go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

Create `apps/desktop/frontend/dist/index.html` (placeholder until Task 18): `<!doctype html><title>Tiles Spliter</title><p>prefs UI placeholder</p>`

`apps/desktop/main.go`:
```go
package main

import (
	"embed"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/app"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

//go:embed all:frontend/dist
var assets embed.FS

// Menu order mirrors the original Tiles tray menu.
var menuActions = []engine.Action{
	engine.ActionCenter, engine.ActionFullscreen,
	engine.ActionHalfLeft, engine.ActionHalfRight, engine.ActionHalfTop, engine.ActionHalfBottom,
	engine.ActionUpperLeft, engine.ActionUpperRight, engine.ActionLowerLeft, engine.ActionLowerRight,
	engine.ActionNextThird, engine.ActionPrevThird,
	engine.ActionTwoThirdsLeft, engine.ActionTwoThirdsRight, engine.ActionTwoThirdsCenter,
	engine.ActionNextDisplay, engine.ActionPrevDisplay,
}

var actionLabels = map[engine.Action]string{
	engine.ActionCenter: "Center", engine.ActionFullscreen: "Fullscreen",
	engine.ActionHalfLeft: "Half Left", engine.ActionHalfRight: "Half Right",
	engine.ActionHalfTop: "Half Top", engine.ActionHalfBottom: "Half Bottom",
	engine.ActionUpperLeft: "Upper Left", engine.ActionUpperRight: "Upper Right",
	engine.ActionLowerLeft: "Lower Left", engine.ActionLowerRight: "Lower Right",
	engine.ActionNextThird: "Next Third", engine.ActionPrevThird: "Previous Third",
	engine.ActionTwoThirdsLeft: "Two Thirds Left", engine.ActionTwoThirdsRight: "Two Thirds Right",
	engine.ActionTwoThirdsCenter: "Two Thirds Center",
	engine.ActionNextDisplay: "Next Display", engine.ActionPrevDisplay: "Previous Display",
}

func main() {
	store := app.NewSettingsStore() // loads from Application Support
	dispatcher := app.NewDispatcher(app.RealPlatform{}, store.Get)
	svc := app.NewSettingsService(store, dispatcher)

	wailsApp := application.New(application.Options{
		Name: "Tiles Spliter",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.guilhermevozniak.tilespliter",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				svc.ShowPreferences()
			},
		},
	})
	svc.SetApp(wailsApp)

	// --- Prefs window (hidden until requested) ---
	prefs := wailsApp.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:  "Tiles Spliter",
		Width:  640,
		Height: 520,
		Hidden: true,
	})
	svc.SetPrefsWindow(prefs)

	// --- Tray ---
	tray := wailsApp.NewSystemTray()
	menu := wailsApp.NewMenu()
	menu.Add("Preferences…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { svc.ShowPreferences() })
	menu.AddSeparator()
	for _, a := range menuActions {
		action := a
		menu.Add(actionLabels[action]).OnClick(func(*application.Context) { dispatcher.Perform(action) })
	}
	menu.AddSeparator()
	menu.Add("Undo").OnClick(func(*application.Context) { dispatcher.Undo() })
	menu.AddSeparator()
	menu.Add("About Tiles Spliter").OnClick(func(*application.Context) { svc.ShowPreferences() })
	menu.Add("Quit Tiles Spliter").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { wailsApp.Quit() })
	tray.SetMenu(menu)
	tray.SetLabel("◧") // placeholder glyph; replaced by template icon in Task 23
	svc.SetTray(tray)

	// --- Engine wiring (only once AX permission exists) ---
	wailsApp.OnApplicationEvent("ApplicationDidFinishLaunching", func(*application.ApplicationEvent) {
		if platform.AXTrusted(false) {
			svc.StartEngine()
		} else {
			svc.ShowPreferences() // onboarding view will ask for permission
		}
	})

	if err := wailsApp.Run(); err != nil {
		slog.Error("app exited", "err", err)
	}
}
```
(API-drift note from Global Constraints applies: event name constant may be `events.Mac.ApplicationDidFinishLaunching` in the current alpha — adapt names, keep behavior.)

- [ ] **Step 2: Implement the settings store + service + real platform**

`apps/desktop/internal/app/realplatform_darwin.go`:
```go
//go:build darwin

package app

import (
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

// RealPlatform adapts the cgo platform package to the Platform interface.
type RealPlatform struct{}

func (RealPlatform) FocusedWindow() (AppWindow, error) {
	w, err := platform.FocusedWindow()
	return realWindow{w}, err
}
func (RealPlatform) WindowAt(p engine.Point) (AppWindow, error) {
	w, err := platform.WindowAt(p)
	return realWindow{w}, err
}
func (RealPlatform) Displays() []engine.Display { return platform.Displays() }
func (RealPlatform) ShowOverlay(r engine.Rect)  { platform.ShowOverlay(r) }
func (RealPlatform) HideOverlay()               { platform.HideOverlay() }

type realWindow struct{ w *platform.Window }

func (r realWindow) WinID() uint32                 { if r.w == nil { return 0 }; return r.w.ID }
func (r realWindow) Frame() (engine.Rect, error)   { return r.w.Frame() }
func (r realWindow) SetFrame(f engine.Rect) error  { return r.w.SetFrame(f) }
func (r realWindow) Release()                      { if r.w != nil { r.w.Release() } }
```

`apps/desktop/internal/app/settings_service.go`:
```go
package app

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SettingsStore holds the live settings with thread-safe access.
type SettingsStore struct {
	mu   sync.RWMutex
	s    engine.Settings
	path string
}

func NewSettingsStore() *SettingsStore {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library", "Application Support", "tiles-spliter", "settings.json")
	return &SettingsStore{s: engine.LoadSettings(path), path: path}
}

func (st *SettingsStore) Get() engine.Settings {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s
}

func (st *SettingsStore) Set(s engine.Settings) error {
	st.mu.Lock()
	st.s = s
	st.mu.Unlock()
	return s.Save(st.path)
}

// SettingsService is bound to the frontend via Wails.
type SettingsService struct {
	store      *SettingsStore
	dispatcher *Dispatcher
	app        *application.App
	prefs      *application.WebviewWindow
	tray       *application.SystemTray
	engineOn   bool
}

func NewSettingsService(store *SettingsStore, d *Dispatcher) *SettingsService {
	return &SettingsService{store: store, dispatcher: d}
}

func (s *SettingsService) SetApp(a *application.App)                 { s.app = a }
func (s *SettingsService) SetPrefsWindow(w *application.WebviewWindow) { s.prefs = w }
func (s *SettingsService) SetTray(t *application.SystemTray)         { s.tray = t }

func (s *SettingsService) ShowPreferences() {
	platform.ActivatePrefs()
	s.prefs.Show()
	s.prefs.Focus()
}

// StartEngine registers hotkeys and the drag tap. Idempotent.
func (s *SettingsService) StartEngine() {
	if s.engineOn {
		return
	}
	s.engineOn = true
	s.applyHotkeys(s.store.Get())
	_ = platform.StartDragTap(func(kind platform.DragEventKind, x, y float64, mod bool) {
		s.dispatcher.OnDragEvent(int(kind), x, y, mod)
	})
}

// Hotkey id == index into menuOrder(); undo uses undoHotkeyID.
func menuOrder() []engine.Action {
	return []engine.Action{
		engine.ActionCenter, engine.ActionFullscreen,
		engine.ActionHalfLeft, engine.ActionHalfRight, engine.ActionHalfTop, engine.ActionHalfBottom,
		engine.ActionUpperLeft, engine.ActionUpperRight, engine.ActionLowerLeft, engine.ActionLowerRight,
		engine.ActionNextThird, engine.ActionPrevThird,
		engine.ActionTwoThirdsLeft, engine.ActionTwoThirdsRight, engine.ActionTwoThirdsCenter,
		engine.ActionNextDisplay, engine.ActionPrevDisplay,
	}
}

const undoHotkeyID = 1000

func (s *SettingsService) applyHotkeys(cfg engine.Settings) {
	platform.UnregisterAllHotkeys()
	if !cfg.Hotkeys.Enabled {
		return
	}
	platform.InstallHotkeyHandler(func(id uint32) {
		if id == undoHotkeyID {
			s.dispatcher.Undo()
			return
		}
		actions := menuOrder()
		if int(id) < len(actions) {
			s.dispatcher.Perform(actions[id])
		}
	})
	for i, a := range menuOrder() {
		if hk, ok := cfg.Hotkeys.Bindings[a]; ok {
			_ = platform.RegisterHotkey(uint32(i), hk)
		}
	}
	// Undo binding (⌥⌘Y by default) is stored under a pseudo-action key "undo".
	if hk, ok := cfg.Hotkeys.Bindings[engine.Action("undo")]; ok {
		_ = platform.RegisterHotkey(undoHotkeyID, hk)
	}
}

// --- Frontend-bound API ---

func (s *SettingsService) Get() engine.Settings { return s.store.Get() }

func (s *SettingsService) Update(next engine.Settings) error {
	old := s.store.Get()
	if err := s.store.Set(next); err != nil {
		return err
	}
	s.applySideEffects(old, next)
	if s.app != nil {
		s.app.EmitEvent("settings:changed", next)
	}
	return nil
}

func (s *SettingsService) RestoreDefaultHotkeys() engine.Settings {
	cur := s.store.Get()
	cur.Hotkeys = engine.DefaultSettings().Hotkeys
	_ = s.Update(cur)
	return cur
}

func (s *SettingsService) AXTrusted() bool           { return platform.AXTrusted(false) }
func (s *SettingsService) RequestAXPermission() bool {
	ok := platform.AXTrusted(true)
	if ok {
		s.StartEngine()
	}
	return ok
}

func (s *SettingsService) applySideEffects(old, next engine.Settings) {
	if s.engineOn {
		s.applyHotkeys(next)
	}
	if old.General.LaunchAtLogin != next.General.LaunchAtLogin {
		_ = platform.SetLoginItem(next.General.LaunchAtLogin)
	}
	if s.tray != nil && old.General.ShowMenuBarIcon != next.General.ShowMenuBarIcon {
		if next.General.ShowMenuBarIcon {
			s.tray.Show()
		} else {
			s.tray.Hide()
		}
	}
}
```

Also add `"undo": {keyCode: 16, modifiers: 2304}` (⌥⌘Y — 16=Y, 256+2048) to BOTH default binding tables: in `engine.DefaultSettings()` add `engine.Action("undo"): {keyY, modCmd + modOpt}` and in `packages/shared` `DEFAULT_SETTINGS.hotkeys.bindings` add `undo: b(KEY.Y, MOD.cmd + MOD.opt)`; update the count assertions (17 → 18) in `settings_test.go` (`len(s.Hotkeys.Bindings) != 18`) and shared tests.

- [ ] **Step 3: Verify it runs**

Run: `cd apps/desktop && go test ./... && wails3 dev` (or `go run .` if `wails3 dev` needs config — create minimal `wails.json` with `{"name": "tiles-spliter", "frontend:dir": "frontend"}` style config only if the CLI demands it).
Expected: menu-bar glyph appears; tray menu lists Preferences…, 17 actions, Undo, About, Quit; with Accessibility granted, ⌥⌘← snaps the focused window left; dragging a window to the left screen edge shows the overlay and snaps on drop; ⌘, opens the placeholder prefs window.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat(app): Wails v3 bootstrap with systray, hotkeys, drag snapping, settings service"
```

---

### Task 17: Frontend — scaffold, settings hook, tab shell, General tab

**Files:**
- Create: `apps/desktop/frontend/package.json`, `vite.config.ts`, `tsconfig.json`, `index.html`, `src/main.tsx`, `src/App.tsx`, `src/api.ts`, `src/useSettings.ts`, `src/useSettings.test.tsx`, `src/tabs/GeneralTab.tsx`, `src/index.css`

**Interfaces:**
- Consumes: `@tiles-spliter/shared` types; Wails-generated bindings (`wails3 generate bindings`) wrapped by `src/api.ts`
- Produces:
  - `src/api.ts` — the ONLY module touching Wails bindings (components stay testable): `getSettings(): Promise<Settings>`, `updateSettings(s: Settings): Promise<void>`, `restoreDefaultHotkeys(): Promise<Settings>`, `axTrusted(): Promise<boolean>`, `requestAXPermission(): Promise<boolean>`. Each function calls the generated `SettingsService` binding; export an `setApiForTests(mock)` override.
  - `useSettings()` hook → `{ settings: Settings | null, patch(updater: (s: Settings) => Settings): void }` — optimistic local update + `updateSettings` call
  - Tab shell in `App.tsx`: General / Hotkeys / Snap to Edges / About (stubs for 18–20)

- [ ] **Step 1: Scaffold**

Run from `apps/desktop/frontend`:
```bash
bun create vite@latest . --template react-ts
bun add tailwindcss @tailwindcss/vite
bun add -d vitest @testing-library/react @testing-library/jest-dom jsdom
```
Add `"@tiles-spliter/shared": "workspace:*"` to dependencies and set `"test": "vitest run"`, `"lint": "biome check src"` scripts. `vite.config.ts` uses `react()` + `tailwindcss()` plugins and `test: { environment: "jsdom" }`. `src/index.css` is `@import "tailwindcss";` plus:
```css
:root { color-scheme: dark; }
body { @apply bg-zinc-950 text-zinc-100 antialiased select-none; }
```

- [ ] **Step 2: Write failing hook test**

`src/useSettings.test.tsx`:
```tsx
import { act, renderHook, waitFor } from "@testing-library/react";
import { DEFAULT_SETTINGS } from "@tiles-spliter/shared";
import { describe, expect, it, vi } from "vitest";
import { setApiForTests } from "./api";
import { useSettings } from "./useSettings";

describe("useSettings", () => {
  it("loads settings then patches optimistically and persists", async () => {
    const update = vi.fn().mockResolvedValue(undefined);
    setApiForTests({ getSettings: async () => structuredClone(DEFAULT_SETTINGS), updateSettings: update });
    const { result } = renderHook(() => useSettings());
    await waitFor(() => expect(result.current.settings).not.toBeNull());
    act(() => result.current.patch((s) => ({ ...s, general: { ...s.general, windowPadding: 8 } })));
    expect(result.current.settings?.general.windowPadding).toBe(8);
    await waitFor(() => expect(update).toHaveBeenCalledOnce());
    expect(update.mock.calls[0][0].general.windowPadding).toBe(8);
  });
});
```

Run: `bunx vitest run` — Expected: FAIL (modules missing).

- [ ] **Step 3: Implement api, hook, shell, General tab**

`src/api.ts`:
```ts
import type { Settings } from "@tiles-spliter/shared";

// Generated by `wails3 generate bindings` into ./bindings (gitignored).
// Loaded lazily so tests and plain-browser dev don't explode.
type Api = {
  getSettings: () => Promise<Settings>;
  updateSettings: (s: Settings) => Promise<void>;
  restoreDefaultHotkeys?: () => Promise<Settings>;
  axTrusted?: () => Promise<boolean>;
  requestAXPermission?: () => Promise<boolean>;
};

let override: Partial<Api> | null = null;
export function setApiForTests(mock: Partial<Api>) {
  override = mock;
}

async function bindings() {
  return await import("../bindings/github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/app/settingsservice");
}

export const getSettings: Api["getSettings"] = async () =>
  override?.getSettings ? override.getSettings() : (await bindings()).Get();
export const updateSettings: Api["updateSettings"] = async (s) =>
  override?.updateSettings ? override.updateSettings(s) : (await bindings()).Update(s);
export const restoreDefaultHotkeys = async (): Promise<Settings> =>
  override?.restoreDefaultHotkeys ? override.restoreDefaultHotkeys() : (await bindings()).RestoreDefaultHotkeys();
export const axTrusted = async (): Promise<boolean> =>
  override?.axTrusted ? override.axTrusted() : (await bindings()).AXTrusted();
export const requestAXPermission = async (): Promise<boolean> =>
  override?.requestAXPermission ? override.requestAXPermission() : (await bindings()).RequestAXPermission();
```
(The exact generated path appears after running `wails3 generate bindings` in Task 16's app — adjust the import to the real generated file name.)

`src/useSettings.ts`:
```ts
import { useCallback, useEffect, useState } from "react";
import type { Settings } from "@tiles-spliter/shared";
import { getSettings, updateSettings } from "./api";

export function useSettings() {
  const [settings, setSettings] = useState<Settings | null>(null);
  useEffect(() => {
    getSettings().then(setSettings);
  }, []);
  const patch = useCallback((updater: (s: Settings) => Settings) => {
    setSettings((cur) => {
      if (!cur) return cur;
      const next = updater(cur);
      void updateSettings(next);
      return next;
    });
  }, []);
  return { settings, patch };
}
```

`src/App.tsx` — modern custom design: dark window, left icon rail (General ⚙, Hotkeys ⌘, Snap ◰, About ⓘ) using Tailwind; renders the active tab, passing `{settings, patch}`; shows a centered spinner until settings load. Stub components for Hotkeys/Snap/About render `<p>Coming in a later task</p>`.

`src/tabs/GeneralTab.tsx`:
```tsx
import type { Settings } from "@tiles-spliter/shared";

type Props = { settings: Settings; patch: (u: (s: Settings) => Settings) => void };

function Toggle({ label, hint, checked, onChange }: { label: string; hint?: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-start justify-between gap-4 py-3">
      <span>
        <span className="block text-sm font-medium">{label}</span>
        {hint && <span className="block text-xs text-zinc-400">{hint}</span>}
      </span>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="h-5 w-9 accent-indigo-500" role="switch" />
    </label>
  );
}

export function GeneralTab({ settings, patch }: Props) {
  const g = settings.general;
  const set = (k: keyof Settings["general"], v: boolean | number) =>
    patch((s) => ({ ...s, general: { ...s.general, [k]: v } }));
  return (
    <div className="divide-y divide-zinc-800 px-6">
      <Toggle label="Launch Tiles Spliter at login" hint="Start automatically." checked={g.launchAtLogin} onChange={(v) => set("launchAtLogin", v)} />
      <Toggle label="Show in the menu bar" hint="When hidden, relaunch the app from Finder to open Preferences." checked={g.showMenuBarIcon} onChange={(v) => set("showMenuBarIcon", v)} />
      <Toggle label="Enable animations" checked={g.enableAnimations} onChange={(v) => set("enableAnimations", v)} />
      <div className="py-3">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">Windows padding</span>
          <span className="text-xs tabular-nums text-zinc-400">{g.windowPadding} pt</span>
        </div>
        <input type="range" min={0} max={50} value={g.windowPadding} onChange={(e) => set("windowPadding", Number(e.target.value))} className="mt-2 w-full accent-indigo-500" />
        <Toggle label="Enable for fullscreen windows" hint="Apply padding to fullscreen windows too." checked={g.padFullscreen} onChange={(v) => set("padFullscreen", v)} />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run tests + dev build**

Run: `bunx vitest run` — Expected: hook test PASS.
Run: `bun run build` (tsc + vite) — Expected: builds into `dist/`; then `cd .. && wails3 dev` shows the real General tab, and toggling "Enable animations" immediately changes snap behavior (persisted — check `settings.json`).

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(frontend): prefs shell, settings hook, General tab"
```

---

### Task 18: Frontend — Hotkeys tab with capture fields

**Files:**
- Create: `src/tabs/HotkeysTab.tsx`, `src/hotkeyCapture.ts`, `src/hotkeyCapture.test.ts`
- Modify: `src/App.tsx` (swap stub)

**Interfaces:**
- Consumes: `useSettings`, `HOTKEY_DISPLAY`, `ACTION_LABELS`, `MOD` from shared; `restoreDefaultHotkeys` from api
- Produces: `keyEventToHotkey(e: {keyCode?: number; code: string; metaKey: boolean; altKey: boolean; ctrlKey: boolean; shiftKey: boolean}): Hotkey | null` — converts a DOM keydown to a Carbon hotkey (null when no non-modifier key or no modifier held)

- [ ] **Step 1: Write failing test**

`src/hotkeyCapture.test.ts`:
```ts
import { describe, expect, it } from "vitest";
import { keyEventToHotkey } from "./hotkeyCapture";

const ev = (code: string, mods: Partial<{ metaKey: boolean; altKey: boolean; ctrlKey: boolean; shiftKey: boolean }>) =>
  ({ code, metaKey: false, altKey: false, ctrlKey: false, shiftKey: false, ...mods });

describe("keyEventToHotkey", () => {
  it("maps ⌥⌘C", () => {
    expect(keyEventToHotkey(ev("KeyC", { metaKey: true, altKey: true }))).toEqual({ keyCode: 8, modifiers: 256 + 2048 });
  });
  it("maps arrows", () => {
    expect(keyEventToHotkey(ev("ArrowLeft", { ctrlKey: true, altKey: true }))).toEqual({ keyCode: 123, modifiers: 4096 + 2048 });
  });
  it("rejects modifier-only and unmodified keys", () => {
    expect(keyEventToHotkey(ev("MetaLeft", { metaKey: true }))).toBeNull();
    expect(keyEventToHotkey(ev("KeyC", {}))).toBeNull();
  });
});
```
Run: `bunx vitest run hotkeyCapture` — Expected: FAIL.

- [ ] **Step 2: Implement**

`src/hotkeyCapture.ts`:
```ts
import { MOD, type Hotkey } from "@tiles-spliter/shared";

// DOM KeyboardEvent.code → Carbon virtual key code (extend as needed).
const CODE_TO_CARBON: Record<string, number> = {
  KeyA: 0, KeyS: 1, KeyD: 2, KeyF: 3, KeyH: 4, KeyG: 5, KeyZ: 6, KeyX: 7, KeyC: 8, KeyV: 9,
  KeyB: 11, KeyQ: 12, KeyW: 13, KeyE: 14, KeyR: 15, KeyY: 16, KeyT: 17,
  Digit1: 18, Digit2: 19, Digit3: 20, Digit4: 21, Digit6: 22, Digit5: 23, Digit9: 25, Digit7: 26,
  Digit8: 28, Digit0: 29, KeyO: 31, KeyU: 32, KeyI: 34, KeyP: 35, KeyL: 37, KeyJ: 38, KeyK: 40,
  KeyN: 45, KeyM: 46, Space: 49,
  ArrowLeft: 123, ArrowRight: 124, ArrowDown: 125, ArrowUp: 126,
};

export function keyEventToHotkey(e: { code: string; metaKey: boolean; altKey: boolean; ctrlKey: boolean; shiftKey: boolean }): Hotkey | null {
  const keyCode = CODE_TO_CARBON[e.code];
  if (keyCode === undefined) return null;
  let modifiers = 0;
  if (e.metaKey) modifiers += MOD.cmd;
  if (e.shiftKey) modifiers += MOD.shift;
  if (e.altKey) modifiers += MOD.opt;
  if (e.ctrlKey) modifiers += MOD.ctrl;
  if (modifiers === 0) return null;
  return { keyCode, modifiers };
}
```

`src/tabs/HotkeysTab.tsx` — two-column grid of the 18 bindings (17 actions + undo) in the original screenshot's order. Each row: action label + a button showing `HOTKEY_DISPLAY(binding)` (or "—"). Clicking a button focuses "recording" mode (ring highlight, "Type shortcut…"); a keydown runs `keyEventToHotkey`; valid → `patch` the binding (duplicate of another action → red ring + inline "Already used by X" and no patch), Escape cancels, Backspace clears the binding. Footer: master "Enable Hotkeys" toggle (left) and "Restore Defaults…" button (right) calling `restoreDefaultHotkeys` then replacing local state via `patch(() => restored)`.

- [ ] **Step 3: Run tests + manual check**

Run: `bunx vitest run` — Expected: PASS.
Run `wails3 dev`: record a custom hotkey (e.g. ⌥⌘P for Center), verify it fires; verify conflict message; Restore Defaults brings back ⌥⌘C.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat(frontend): hotkeys tab with capture, conflict detection, restore defaults"
```

---

### Task 19: Frontend — Snap to Edges tab

**Files:**
- Create: `src/tabs/SnapTab.tsx`
- Modify: `src/App.tsx` (swap stub)

**Interfaces:**
- Consumes: `useSettings`, `ZONES`, `ACTIONS`, `ACTION_LABELS` from shared
- Produces: complete Snap tab UI bound to `settings.snap`

- [ ] **Step 1: Implement**

`src/tabs/SnapTab.tsx` layout (all bound via `patch`):
1. Toggle "Snap windows when moved to the edges of the screen" (`snap.enabled`) — when off, the rest renders at 40% opacity with `pointer-events-none`.
2. Toggle "Restore previous window size when moved again" (`snap.restorePreviousSize`).
3. **Active Edges** card: a 16:10 rounded rectangle styled as a mini desktop (gradient bg). Eight absolutely-positioned `<select>` dropdowns — corners at the four corners, edge selects centered on each side — one per `ZoneID`, options = all 17 actions + "—" (`none`), value = `snap.zones[zone]`.
4. Hint row: "Hold ⌥ ALT or ⌘ CMD while dragging a window to the left or right edge to snap it to thirds."
5. Slider "Snap zone thickness": 1–40 pt, live value label (`snap.zoneThickness`).
6. Slider "Delay before activation": 0–1000 ms in 50 ms steps, showing "None" at 0 (`snap.activationDelayMs`).

Example zone select (repeated via a `ZONE_POS: Record<ZoneID, string>` map of Tailwind position classes):
```tsx
<select
  value={settings.snap.zones[zone]}
  onChange={(e) => patch((s) => ({ ...s, snap: { ...s.snap, zones: { ...s.snap.zones, [zone]: e.target.value as Action } } }))}
  className={`absolute ${ZONE_POS[zone]} rounded-md bg-zinc-800/90 px-2 py-1 text-xs`}
>
  {ACTIONS.map((a) => (
    <option key={a} value={a}>{ACTION_LABELS[a]}</option>
  ))}
</select>
```

- [ ] **Step 2: Manual check**

Run `wails3 dev`: set the bottom zone to "Half Bottom", drag a window to the bottom edge → it snaps; set thickness to 30 → zones trigger further from the edge; set delay to 500ms → overlay appears only after hovering the edge half a second; disable snapping → nothing triggers.

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat(frontend): snap-to-edges tab with per-zone mapping"
```

---

### Task 20: Frontend — About tab + Accessibility onboarding

**Files:**
- Create: `src/tabs/AboutTab.tsx`, `src/Onboarding.tsx`
- Modify: `src/App.tsx`

**Interfaces:**
- Consumes: `axTrusted`, `requestAXPermission` from api
- Produces: onboarding gate rendered INSTEAD of the tab shell whenever `axTrusted()` is false

- [ ] **Step 1: Implement**

`src/Onboarding.tsx`: centered card — app icon, "Tiles Spliter needs Accessibility access to move and resize windows.", a "Grant Access…" button calling `requestAXPermission()` (macOS shows the system prompt / System Settings deep link), and a 2 s `setInterval` polling `axTrusted()`; when it flips true, call the `onGranted` prop (App re-renders the tab shell — the Go side already started the engine inside `RequestAXPermission`).

`src/tabs/AboutTab.tsx`: app icon, name, version (hardcode `0.1.0`, single source: `import { version } from "../../package.json"`), one-line description, link to the marketing site, MIT-style credits line mentioning Rectangle as prior art inspiration.

`App.tsx`: on mount call `axTrusted()`; `false` → render `<Onboarding onGranted={...}>`, `true` → tab shell.

- [ ] **Step 2: Manual check**

Remove the app from System Settings → Privacy & Security → Accessibility, relaunch → onboarding appears; grant → UI switches to tabs and hotkeys start working without a restart.

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat(frontend): about tab and accessibility onboarding flow"
```

---

### Task 21: Marketing site (`apps/web`)

**Files:**
- Create: `apps/web/package.json`, `next.config.ts`, `tsconfig.json`, `app/layout.tsx`, `app/page.tsx`, `app/globals.css`, `app/privacy/page.tsx`, `app/changelog/page.tsx`, `components/HotkeyTable.tsx`, `components/Hero.tsx`, `e2e/smoke.spec.ts`, `playwright.config.ts`

**Interfaces:**
- Consumes: `@tiles-spliter/shared` (`DEFAULT_SETTINGS`, `ACTION_LABELS`, `HOTKEY_DISPLAY`)
- Produces: static-exported site (`out/`) with landing, privacy, changelog

- [ ] **Step 1: Scaffold**

Run from `apps/web`: `bunx create-next-app@latest . --ts --tailwind --app --no-src-dir --import-alias "@/*"`; add `"@tiles-spliter/shared": "workspace:*"`; `next.config.ts` sets `output: "export"`. Add `"test": "playwright test"`, `"lint": "biome check app components"` scripts; `bun add -d @playwright/test`.

- [ ] **Step 2: Write failing Playwright smoke test**

`e2e/smoke.spec.ts`:
```ts
import { expect, test } from "@playwright/test";

test("landing renders hero, hotkey table, download", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toContainText("Tiles Spliter");
  await expect(page.getByRole("link", { name: /download/i })).toHaveAttribute("href", /github\.com\/.+\/releases/);
  await expect(page.getByText("⌥⌘C")).toBeVisible(); // hotkey table driven by shared defaults
  await expect(page.getByText("Half Left")).toBeVisible();
});
```
`playwright.config.ts` uses `webServer: { command: "bun run build && bunx serve out -l 4123", url: "http://localhost:4123" }` (add `serve` as devDependency).
Run: `bunx playwright test` — Expected: FAIL (default create-next-app page).

- [ ] **Step 3: Implement pages**

- `app/layout.tsx`: dark theme, Inter font, metadata (title "Tiles Spliter — window manager for macOS", description), simple header (logo left; Changelog / Privacy / GitHub links right), footer.
- `components/Hero.tsx`: h1 "Tiles Spliter", tagline "Snap, tile and arrange your Mac windows — by drag, hotkey or menu bar.", Download button linking `https://github.com/GuilhermeVozniak/tiles-spliter/releases/latest`, "Requires macOS 13+" small print, and a pure-CSS animated demo: a mini desktop `div` where two `div` "windows" animate into half-left/half-right positions on a keyframe loop (no JS, ~30 lines of Tailwind + a `@keyframes` in `globals.css`).
- `app/page.tsx`: Hero + three-card feature grid (Snap to Edges / Hotkeys for everything / Multi-display) + `<HotkeyTable />` + a final CTA repeat of the download button.
- `components/HotkeyTable.tsx`:
```tsx
import { ACTION_LABELS, DEFAULT_SETTINGS, HOTKEY_DISPLAY } from "@tiles-spliter/shared";

export function HotkeyTable() {
  const entries = Object.entries(DEFAULT_SETTINGS.hotkeys.bindings);
  return (
    <table className="mx-auto text-sm">
      <tbody>
        {entries.map(([action, hk]) => (
          <tr key={action} className="border-b border-zinc-800">
            <td className="py-2 pr-8">{ACTION_LABELS[action as keyof typeof ACTION_LABELS] ?? action}</td>
            <td className="py-2 font-mono text-zinc-400">{HOTKEY_DISPLAY(hk)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```
- `app/privacy/page.tsx`: "Tiles Spliter runs entirely on your Mac. No analytics, no network calls, no data collection. Accessibility permission is used only to move and resize windows."
- `app/changelog/page.tsx`: `0.1.0 — initial release` list.

- [ ] **Step 4: Run test to verify it passes**

Run: `bunx playwright test` — Expected: PASS. Also `bun run build` produces `out/`.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat(web): marketing site with shared-driven hotkey table"
```

---

### Task 22: Packaging — app bundle, icon, signing, notarization, DMG

**Files:**
- Create: `apps/desktop/build/Info.plist`, `apps/desktop/build/entitlements.plist`, `apps/desktop/build/icon.iconset/` (generated), `apps/desktop/build/appicon.png` (1024×1024), `scripts/release.sh`, `apps/desktop/build/tray-icon.png` (template image, 22×22 black+alpha)
- Modify: `apps/desktop/main.go` (tray icon instead of label glyph)

**Interfaces:**
- Consumes: the built binary from Task 16
- Produces: `dist/Tiles Spliter-<version>.dmg`, signed + notarized. Requires env: `CODESIGN_IDENTITY` ("Developer ID Application: …"), `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_PASSWORD` (app-specific password for notarytool).

- [ ] **Step 1: Create bundle metadata**

`apps/desktop/build/Info.plist` — keys: `CFBundleName`/`CFBundleDisplayName` "Tiles Spliter", `CFBundleIdentifier` `com.guilhermevozniak.tilespliter`, `CFBundleShortVersionString`/`CFBundleVersion` `0.1.0`, `LSMinimumSystemVersion` `13.0`, `LSUIElement` `true` (menu-bar app: no Dock icon at launch; prefs focus is handled by the activation-policy switch from Task 14), `NSHighResolutionCapable` `true`, `CFBundleIconFile` `icon`.

`apps/desktop/build/entitlements.plist`: hardened-runtime-compatible — empty `<dict/>` is correct here (no sandbox: AX apps can't be sandboxed; no special entitlements needed for AX).

App icon: generate `icon.icns` from `appicon.png` via `iconutil` (sips resize into `icon.iconset`, then `iconutil -c icns`). Draw a simple 2-tile rounded-square icon (two rectangles side by side) — any vector tool or a quick SVG rendered with `qlmanage`/`rsvg-convert`.

Tray icon: 22×22 (and @2x 44×44) black-with-alpha PNG of two side-by-side tiles; in `main.go` replace `tray.SetLabel("◧")` with `tray.SetTemplateIcon(trayIconBytes)` (`//go:embed build/tray-icon.png`) so it adapts to menu-bar light/dark.

- [ ] **Step 2: Write the release script**

`scripts/release.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' apps/desktop/build/Info.plist)
APP="dist/Tiles Spliter.app"

# 1. Frontends
(cd apps/desktop/frontend && bun install && bun run build)

# 2. Universal binary
(cd apps/desktop && GOOS=darwin GOARCH=arm64 go build -o ../../dist/tilespliter-arm64 . \
  && GOOS=darwin GOARCH=amd64 go build -o ../../dist/tilespliter-amd64 .)
lipo -create -output dist/tilespliter dist/tilespliter-arm64 dist/tilespliter-amd64

# 3. Bundle
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp apps/desktop/build/Info.plist "$APP/Contents/"
cp dist/tilespliter "$APP/Contents/MacOS/Tiles Spliter"
cp apps/desktop/build/icon.icns "$APP/Contents/Resources/"

# 4. Sign (hardened runtime)
codesign --force --options runtime --timestamp \
  --entitlements apps/desktop/build/entitlements.plist \
  --sign "$CODESIGN_IDENTITY" "$APP"

# 5. Notarize + staple
ditto -c -k --keepParent "$APP" dist/notarize.zip
xcrun notarytool submit dist/notarize.zip --apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID" --password "$APPLE_APP_PASSWORD" --wait
xcrun stapler staple "$APP"

# 6. DMG
hdiutil create -volname "Tiles Spliter" -srcfolder "$APP" -ov -format UDZO "dist/Tiles Spliter-$VERSION.dmg"
echo "done: dist/Tiles Spliter-$VERSION.dmg"
```
`chmod +x scripts/release.sh`. Note: cross-`GOARCH` builds with cgo need both toolchains; on Apple Silicon `GOARCH=amd64` requires `CGO_ENABLED=1` with `clang -arch x86_64` — if the amd64 leg fails locally, ship arm64-only first (`lipo` step becomes a copy) and revisit; do not block the release on it.

- [ ] **Step 3: Run and verify**

Run: `CODESIGN_IDENTITY="Developer ID Application: <name>" APPLE_ID=gui336699@gmail.com APPLE_TEAM_ID=<team> APPLE_APP_PASSWORD=<app-pw> ./scripts/release.sh`
Expected: DMG in `dist/`; `spctl -a -vv "dist/Tiles Spliter.app"` says "accepted … Notarized Developer ID"; mounting the DMG and launching the app shows the onboarding, and after granting AX everything works.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "build: app bundle, signing, notarization, DMG release script"
```

---

### Task 23: Monorepo wiring + full parity smoke checklist

**Files:**
- Create: `apps/desktop/package.json` (Turbo adapter), `docs/SMOKE.md`
- Modify: root `README.md`

- [ ] **Step 1: Wire desktop into Turbo**

`apps/desktop/package.json`:
```json
{
  "name": "@tiles-spliter/desktop",
  "private": true,
  "scripts": {
    "dev": "wails3 dev",
    "build": "cd frontend && bun run build && cd .. && go build -o bin/tilespliter .",
    "test": "go test ./...",
    "lint": "go vet ./..."
  }
}
```
Run: `bun run test && bun run build && bun run lint` from the repo root — Expected: engine/app/platform Go tests, shared + frontend Vitest, and web build all pass via Turbo.

- [ ] **Step 2: Write the parity smoke checklist**

`docs/SMOKE.md` — manual checklist run before every release (each item checked against a real app like Safari):
1. All 17 tray-menu actions place the window exactly (halves, quarters, thirds cycle L→M→R with wrap, two-thirds ×3, center keeps size, fullscreen fills visible frame).
2. Every default hotkey from the spec table fires its action; Enable Hotkeys off kills all of them; remap + restore defaults works.
3. Drag to each of the 8 zones: overlay previews after the configured delay, drop applies, disabled zone does nothing.
4. ⌥-drag and ⌘-drag to left/right edges snap to thirds.
5. Restore previous size: snap by drag, drag away → original size returns.
6. Undo (⌥⌘Y and menu) restores the pre-action frame repeatedly (stack).
7. Padding 10pt visibly gaps snapped windows; fullscreen padding only when enabled.
8. Two displays: next/prev display moves proportionally; snapping works on the second display; zones on both.
9. Animations toggle changes snap movement between animated/instant.
10. Launch at login registers (System Settings → Login Items); menu-bar icon hide → app keeps working, relaunch from Finder opens Preferences.
11. Kill the settings file with garbage → app launches with defaults, `.bak` created.
12. Non-resizable window (e.g. System Settings) → actions fail silently, no crash.

- [ ] **Step 3: Run the checklist, fix what fails, commit**

```bash
git add -A && git commit -m "chore: turbo wiring and release smoke checklist"
```

---

## Plan Self-Review Notes

- **Spec coverage:** all four preference tabs (Tasks 17–20), 17 actions + undo (3–6, 15–16), snap-to-edges incl. delay/thickness/restore/modifier-thirds/per-zone config (8, 9, 12, 19), hotkeys with defaults/remap/restore (2, 7, 11, 18), multi-display (5), animations + padding (3, 14), login item + headless + single-instance (14, 16), onboarding (20), marketing site (21), signed DMG (22), parity checklist (23). Undo default binding is added in Task 16 Step 2 (bindings count becomes 18).
- **Known alpha risk:** Wails v3 API names in Tasks 16–17 may drift (events, systray, bindings paths) — the Global Constraints adaptation rule applies.
- **Deliberate scope cuts (per spec):** no auto-update, no Homebrew cask, no Windows/Linux.
