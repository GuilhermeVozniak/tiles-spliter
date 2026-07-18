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
  it("rejects shift-only combos (would swallow capital letters system-wide)", () => {
    expect(keyEventToHotkey(ev("KeyY", { shiftKey: true }))).toBeNull();
    expect(keyEventToHotkey(ev("ArrowLeft", { shiftKey: true }))).toBeNull();
  });
  it("accepts shift combined with another modifier", () => {
    expect(
      keyEventToHotkey(ev("KeyY", { shiftKey: true, metaKey: true })),
    ).toEqual({ keyCode: 16, modifiers: 512 + 256 });
    expect(
      keyEventToHotkey(ev("KeyY", { shiftKey: true, altKey: true })),
    ).toEqual({ keyCode: 16, modifiers: 512 + 2048 });
  });
});
