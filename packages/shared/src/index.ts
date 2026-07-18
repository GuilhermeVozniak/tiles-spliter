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
    // "undo" is a pseudo-action: bindable like the others but not part of
    // the Action union (it has no window-layout behavior of its own).
    bindings: Partial<Record<Action, Hotkey>> & { undo?: Hotkey };
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
      undo: b(KEY.Y, MOD.cmd + MOD.opt),
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
