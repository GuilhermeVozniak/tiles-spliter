import { describe, expect, it } from "vitest";
import {
  ACTION_LABELS,
  ACTIONS,
  actionLabel,
  CODE_TO_CARBON,
  DEFAULT_SETTINGS,
  HOTKEY_DISPLAY,
  ZONES,
} from "./index";

describe("shared defaults", () => {
  it("has 17 actions plus none", () => {
    expect(ACTIONS).toHaveLength(18);
    expect(ACTIONS).toContain("center");
    expect(ACTIONS).toContain("none");
  });
  it("has 8 zones with parity defaults", () => {
    expect(ZONES).toHaveLength(8);
    expect(DEFAULT_SETTINGS.snap.zones.left).toBe("half-left");
    expect(DEFAULT_SETTINGS.snap.zones.top).toBe("fullscreen");
    expect(DEFAULT_SETTINGS.snap.zones.bottom).toBe("none");
    expect(DEFAULT_SETTINGS.snap.zoneThickness).toBe(10);
    expect(DEFAULT_SETTINGS.snap.activationDelayMs).toBe(0);
  });
  it("default hotkeys match Tiles (spot checks)", () => {
    // Carbon: cmd=256, opt=2048, ctrl=4096; C=8, left arrow=123
    expect(DEFAULT_SETTINGS.hotkeys.bindings.center).toEqual({
      keyCode: 8,
      modifiers: 256 + 2048,
    });
    expect(DEFAULT_SETTINGS.hotkeys.bindings["half-left"]).toEqual({
      keyCode: 123,
      modifiers: 256 + 2048,
    });
    expect(DEFAULT_SETTINGS.hotkeys.bindings["next-display"]).toEqual({
      keyCode: 124,
      modifiers: 256 + 2048 + 4096,
    });
  });
  it("formats hotkeys for display", () => {
    expect(HOTKEY_DISPLAY({ keyCode: 8, modifiers: 256 + 2048 })).toBe("⌥⌘C");
  });
  it("formats a recorded ⌥⌘O hotkey (regression: keyCode 31 used to show as #31)", () => {
    expect(HOTKEY_DISPLAY({ keyCode: 31, modifiers: 2304 })).toBe("⌥⌘O");
  });
  it("has a display name for every capturable key code", () => {
    for (const code of Object.values(CODE_TO_CARBON)) {
      const label = HOTKEY_DISPLAY({ keyCode: code, modifiers: 256 });
      expect(label).not.toContain("#");
    }
  });
  it("labels every action", () => {
    for (const a of ACTIONS) expect(ACTION_LABELS[a]).toBeTruthy();
  });
});

describe("actionLabel", () => {
  it("labels every real action the same as ACTION_LABELS", () => {
    for (const a of ACTIONS) expect(actionLabel(a)).toBe(ACTION_LABELS[a]);
  });
  it('labels the "undo" pseudo-action', () => {
    expect(actionLabel("undo")).toBe("Undo");
  });
  it("falls back to a title-cased rendering for unknown keys", () => {
    expect(actionLabel("some-unknown-action")).toBe("Some Unknown Action");
  });
});
