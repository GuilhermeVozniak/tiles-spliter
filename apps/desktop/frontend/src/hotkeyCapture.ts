import { CODE_TO_CARBON, type Hotkey, MOD } from "@tiles-spliter/shared";

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
  // Reject unmodified keys and shift-only combos: registering e.g. ⇧Y
  // system-wide would swallow every capital "Y" keystroke everywhere.
  // Shift is fine when combined with cmd/opt/ctrl.
  if (modifiers === 0 || modifiers === MOD.shift) return null;
  return { keyCode, modifiers };
}
