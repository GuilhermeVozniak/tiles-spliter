import {
  ACTION_LABELS,
  type Action,
  HOTKEY_DISPLAY,
  type Hotkey,
  type Settings,
} from "@tiles-spliter/shared";
import { useCallback, useEffect, useState } from "react";
import { restoreDefaultHotkeys } from "../api";
import { keyEventToHotkey } from "../hotkeyCapture";

type Props = {
  settings: Settings;
  patch: (u: (s: Settings) => Settings) => void;
};

type BindingKey = Action | "undo";

// Two-column grid, screenshot order: layout actions left-to-right, top-to-bottom.
const ORDER: BindingKey[] = [
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
  "undo",
];

function labelFor(key: BindingKey): string {
  return key === "undo" ? "Undo" : ACTION_LABELS[key];
}

function findConflict(
  bindings: Partial<Record<BindingKey, Hotkey>>,
  key: BindingKey,
  hotkey: Hotkey,
): BindingKey | null {
  for (const [otherKey, otherHotkey] of Object.entries(bindings) as [
    BindingKey,
    Hotkey | undefined,
  ][]) {
    if (otherKey === key || !otherHotkey) continue;
    if (
      otherHotkey.keyCode === hotkey.keyCode &&
      otherHotkey.modifiers === hotkey.modifiers
    ) {
      return otherKey;
    }
  }
  return null;
}

function HotkeyButton({
  hotkey,
  recording,
  conflictWith,
  onStartRecording,
}: {
  hotkey: Hotkey | undefined;
  recording: boolean;
  conflictWith: BindingKey | null;
  onStartRecording: () => void;
}) {
  const hasConflict = recording && conflictWith !== null;
  return (
    <div className="flex flex-col items-end gap-1">
      <button
        type="button"
        onClick={onStartRecording}
        className={`min-w-[7rem] rounded-md border px-3 py-1.5 text-right text-sm tabular-nums transition-colors ${
          hasConflict
            ? "border-red-500 bg-red-500/10 text-red-300 ring-2 ring-red-500"
            : recording
              ? "border-indigo-500 bg-indigo-500/10 text-indigo-200 ring-2 ring-indigo-500"
              : "border-zinc-700 bg-zinc-900 text-zinc-200 hover:border-zinc-600 hover:bg-zinc-800"
        }`}
      >
        {recording ? "Type shortcut…" : hotkey ? HOTKEY_DISPLAY(hotkey) : "—"}
      </button>
      {hasConflict && conflictWith && (
        <span className="text-xs text-red-400">
          Already used by {labelFor(conflictWith)}
        </span>
      )}
    </div>
  );
}

export function HotkeysTab({ settings, patch }: Props) {
  const [recording, setRecording] = useState<BindingKey | null>(null);
  const [conflict, setConflict] = useState<BindingKey | null>(null);
  const bindings = settings.hotkeys.bindings;

  const stopRecording = useCallback(() => {
    setRecording(null);
    setConflict(null);
  }, []);

  useEffect(() => {
    if (!recording) return;

    const onKeyDown = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();

      if (e.code === "Escape") {
        stopRecording();
        return;
      }

      if (e.code === "Backspace" || e.code === "Delete") {
        patch((s) => {
          const next = { ...s.hotkeys.bindings };
          delete next[recording];
          return { ...s, hotkeys: { ...s.hotkeys, bindings: next } };
        });
        stopRecording();
        return;
      }

      const hotkey = keyEventToHotkey(e);
      if (!hotkey) return;

      const conflictKey = findConflict(bindings, recording, hotkey);
      if (conflictKey) {
        setConflict(conflictKey);
        return;
      }

      patch((s) => ({
        ...s,
        hotkeys: {
          ...s.hotkeys,
          bindings: { ...s.hotkeys.bindings, [recording]: hotkey },
        },
      }));
      stopRecording();
    };

    window.addEventListener("keydown", onKeyDown, true);
    return () => window.removeEventListener("keydown", onKeyDown, true);
  }, [recording, bindings, patch, stopRecording]);

  const left = ORDER.slice(0, 9);
  const right = ORDER.slice(9);

  return (
    <div className="flex h-full flex-col px-6">
      <div className="flex-1 overflow-y-auto">
        <div className="grid grid-cols-2 gap-x-8 gap-y-1">
          {(
            [
              ["left", left],
              ["right", right],
            ] as const
          ).map(([colKey, col]) => (
            <div
              key={colKey}
              className="flex flex-col divide-y divide-zinc-800"
            >
              {col.map((key) => (
                <div
                  key={key}
                  className="flex items-center justify-between gap-4 py-2.5"
                >
                  <span className="text-sm text-zinc-200">{labelFor(key)}</span>
                  <HotkeyButton
                    hotkey={bindings[key]}
                    recording={recording === key}
                    conflictWith={recording === key ? conflict : null}
                    onStartRecording={() => {
                      setConflict(null);
                      setRecording(key);
                    }}
                  />
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-zinc-800 pt-4">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            role="switch"
            aria-checked={settings.hotkeys.enabled}
            checked={settings.hotkeys.enabled}
            onChange={(e) =>
              patch((s) => ({
                ...s,
                hotkeys: { ...s.hotkeys, enabled: e.target.checked },
              }))
            }
            className="h-5 w-9 accent-indigo-500"
          />
          <span className="text-sm font-medium">Enable Hotkeys</span>
        </label>
        <button
          type="button"
          onClick={async () => {
            const restored = await restoreDefaultHotkeys();
            patch(() => restored);
          }}
          className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-1.5 text-sm text-zinc-200 hover:border-zinc-600 hover:bg-zinc-800"
        >
          Restore Defaults…
        </button>
      </div>
    </div>
  );
}
