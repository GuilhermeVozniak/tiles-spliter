import { describe, expect, it } from "vitest";
import { keyEventToHotkey } from "./hotkeyCapture";

const ev = (
  code: string,
  mods: Partial<{
    metaKey: boolean;
    altKey: boolean;
    ctrlKey: boolean;
    shiftKey: boolean;
  }>,
) => ({
  code,
  metaKey: false,
  altKey: false,
  ctrlKey: false,
  shiftKey: false,
  ...mods,
});

describe("keyEventToHotkey", () => {
  it("maps ⌥⌘C", () => {
    expect(
      keyEventToHotkey(ev("KeyC", { metaKey: true, altKey: true })),
    ).toEqual({ keyCode: 8, modifiers: 256 + 2048 });
  });
  it("maps arrows", () => {
    expect(
      keyEventToHotkey(ev("ArrowLeft", { ctrlKey: true, altKey: true })),
    ).toEqual({ keyCode: 123, modifiers: 4096 + 2048 });
  });
  it("rejects modifier-only and unmodified keys", () => {
    expect(keyEventToHotkey(ev("MetaLeft", { metaKey: true }))).toBeNull();
    expect(keyEventToHotkey(ev("KeyC", {}))).toBeNull();
  });
});
