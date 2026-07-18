export const ACTIONS = [
  "center",
  "fullscreen",
  "half-left",
  "half-right",
  "half-top",
  "half-bottom",
  "upper-left",
  "upper-right",
  "lower-left",
  "lower-right",
  "next-third",
  "prev-third",
  "two-thirds-left",
  "two-thirds-right",
  "two-thirds-center",
  "next-display",
  "prev-display",
  "none",
] as const;
export type Action = (typeof ACTIONS)[number];

export const ZONES = [
  "top-left",
  "top",
  "top-right",
  "left",
  "right",
  "bottom-left",
  "bottom",
  "bottom-right",
] as const;
export type ZoneID = (typeof ZONES)[number];

export interface Hotkey {
  keyCode: number;
  modifiers: number;
}

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
  A: 0,
  S: 1,
  D: 2,
  F: 3,
  H: 4,
  G: 5,
  Z: 6,
  X: 7,
  C: 8,
  V: 9,
  B: 11,
  Q: 12,
  W: 13,
  E: 14,
  R: 15,
  Y: 16,
  T: 17,
  digit1: 18,
  digit2: 19,
  digit3: 20,
  digit4: 21,
  digit6: 22,
  digit5: 23,
  digit9: 25,
  digit7: 26,
  digit8: 28,
  digit0: 29,
  O: 31,
  U: 32,
  I: 34,
  P: 35,
  L: 37,
  J: 38,
  K: 40,
  N: 45,
  M: 46,
  space: 49,
  left: 123,
  right: 124,
  down: 125,
  up: 126,
} as const;

// DOM KeyboardEvent.code → Carbon virtual key code, for every key the
// hotkey recorder accepts. Shared so the frontend's capture logic and the
// display formatter below stay in sync (see hotkeyCapture.ts).
export const CODE_TO_CARBON: Record<string, number> = {
  KeyA: KEY.A,
  KeyS: KEY.S,
  KeyD: KEY.D,
  KeyF: KEY.F,
  KeyH: KEY.H,
  KeyG: KEY.G,
  KeyZ: KEY.Z,
  KeyX: KEY.X,
  KeyC: KEY.C,
  KeyV: KEY.V,
  KeyB: KEY.B,
  KeyQ: KEY.Q,
  KeyW: KEY.W,
  KeyE: KEY.E,
  KeyR: KEY.R,
  KeyY: KEY.Y,
  KeyT: KEY.T,
  Digit1: KEY.digit1,
  Digit2: KEY.digit2,
  Digit3: KEY.digit3,
  Digit4: KEY.digit4,
  Digit6: KEY.digit6,
  Digit5: KEY.digit5,
  Digit9: KEY.digit9,
  Digit7: KEY.digit7,
  Digit8: KEY.digit8,
  Digit0: KEY.digit0,
  KeyO: KEY.O,
  KeyU: KEY.U,
  KeyI: KEY.I,
  KeyP: KEY.P,
  KeyL: KEY.L,
  KeyJ: KEY.J,
  KeyK: KEY.K,
  KeyN: KEY.N,
  KeyM: KEY.M,
  Space: KEY.space,
  ArrowLeft: KEY.left,
  ArrowRight: KEY.right,
  ArrowDown: KEY.down,
  ArrowUp: KEY.up,
};

const b = (keyCode: number, modifiers: number): Hotkey => ({
  keyCode,
  modifiers,
});

export const DEFAULT_SETTINGS: Settings = {
  general: {
    launchAtLogin: true,
    showMenuBarIcon: true,
    enableAnimations: true,
    windowPadding: 0,
    padFullscreen: false,
  },
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
      "top-left": "upper-left",
      top: "fullscreen",
      "top-right": "upper-right",
      left: "half-left",
      right: "half-right",
      "bottom-left": "lower-left",
      bottom: "none",
      "bottom-right": "lower-right",
    },
    zoneThickness: 10,
    activationDelayMs: 0,
  },
};

export const ACTION_LABELS: Record<Action, string> = {
  center: "Center",
  fullscreen: "Fullscreen",
  "half-left": "Half Left",
  "half-right": "Half Right",
  "half-top": "Half Top",
  "half-bottom": "Half Bottom",
  "upper-left": "Upper Left",
  "upper-right": "Upper Right",
  "lower-left": "Lower Left",
  "lower-right": "Lower Right",
  "next-third": "Next Third",
  "prev-third": "Previous Third",
  "two-thirds-left": "Two Thirds Left",
  "two-thirds-right": "Two Thirds Right",
  "two-thirds-center": "Two Thirds Center",
  "next-display": "Next Display",
  "prev-display": "Previous Display",
  none: "—",
};

const KEY_NAMES: Record<number, string> = {
  [KEY.A]: "A",
  [KEY.B]: "B",
  [KEY.C]: "C",
  [KEY.D]: "D",
  [KEY.E]: "E",
  [KEY.F]: "F",
  [KEY.G]: "G",
  [KEY.H]: "H",
  [KEY.I]: "I",
  [KEY.J]: "J",
  [KEY.K]: "K",
  [KEY.L]: "L",
  [KEY.M]: "M",
  [KEY.N]: "N",
  [KEY.O]: "O",
  [KEY.P]: "P",
  [KEY.Q]: "Q",
  [KEY.R]: "R",
  [KEY.S]: "S",
  [KEY.T]: "T",
  [KEY.U]: "U",
  [KEY.V]: "V",
  [KEY.W]: "W",
  [KEY.X]: "X",
  [KEY.Y]: "Y",
  [KEY.Z]: "Z",
  [KEY.digit0]: "0",
  [KEY.digit1]: "1",
  [KEY.digit2]: "2",
  [KEY.digit3]: "3",
  [KEY.digit4]: "4",
  [KEY.digit5]: "5",
  [KEY.digit6]: "6",
  [KEY.digit7]: "7",
  [KEY.digit8]: "8",
  [KEY.digit9]: "9",
  [KEY.space]: "Space",
  [KEY.left]: "←",
  [KEY.right]: "→",
  [KEY.down]: "↓",
  [KEY.up]: "↑",
};

export function HOTKEY_DISPLAY(hk: Hotkey): string {
  let s = "";
  if (hk.modifiers & MOD.ctrl) s += "⌃";
  if (hk.modifiers & MOD.opt) s += "⌥";
  if (hk.modifiers & MOD.shift) s += "⇧";
  if (hk.modifiers & MOD.cmd) s += "⌘";
  return s + (KEY_NAMES[hk.keyCode] ?? `#${hk.keyCode}`);
}

// Safe label for an action key: known actions/"undo" use their curated
// label; anything else falls back to a title-cased rendering of the raw
// key instead of leaking the internal kebab-case string to the UI.
export function actionLabel(key: string): string {
  if (key === "undo") return "Undo";
  if (key in ACTION_LABELS) return ACTION_LABELS[key as Action];
  return key
    .split("-")
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}
