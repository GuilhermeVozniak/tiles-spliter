import { type Hotkey, MOD } from "@tiles-spliter/shared";

// DOM KeyboardEvent.code → Carbon virtual key code (extend as needed).
const CODE_TO_CARBON: Record<string, number> = {
  KeyA: 0,
  KeyS: 1,
  KeyD: 2,
  KeyF: 3,
  KeyH: 4,
  KeyG: 5,
  KeyZ: 6,
  KeyX: 7,
  KeyC: 8,
  KeyV: 9,
  KeyB: 11,
  KeyQ: 12,
  KeyW: 13,
  KeyE: 14,
  KeyR: 15,
  KeyY: 16,
  KeyT: 17,
  Digit1: 18,
  Digit2: 19,
  Digit3: 20,
  Digit4: 21,
  Digit6: 22,
  Digit5: 23,
  Digit9: 25,
  Digit7: 26,
  Digit8: 28,
  Digit0: 29,
  KeyO: 31,
  KeyU: 32,
  KeyI: 34,
  KeyP: 35,
  KeyL: 37,
  KeyJ: 38,
  KeyK: 40,
  KeyN: 45,
  KeyM: 46,
  Space: 49,
  ArrowLeft: 123,
  ArrowRight: 124,
  ArrowDown: 125,
  ArrowUp: 126,
};

export function keyEventToHotkey(e: {
  code: string;
  metaKey: boolean;
  altKey: boolean;
  ctrlKey: boolean;
  shiftKey: boolean;
}): Hotkey | null {
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
