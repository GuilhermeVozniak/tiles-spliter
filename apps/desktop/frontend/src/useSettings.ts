import { DEFAULT_SETTINGS, type Settings } from "@tiles-spliter/shared";
import { useCallback, useEffect, useRef, useState } from "react";
import { getSettings, updateSettings } from "./api";

// Trailing debounce: rapid patches (e.g. dragging a slider) collapse into a
// single updateSettings call after the user settles.
const PERSIST_DEBOUNCE_MS = 200;

export function useSettings() {
  const [settings, setSettings] = useState<Settings | null>(null);

  // Chains persistence calls so a later write's promise can never resolve
  // before an earlier one — out-of-order updateSettings calls would let a
  // stale snapshot clobber a newer one on the Go side.
  const persistChain = useRef(Promise.resolve());
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pending = useRef<Settings | null>(null);

  useEffect(() => {
    getSettings()
      .then(setSettings)
      .catch((err) => {
        // Bindings unavailable (tests, plain-browser dev) or the call
        // failed: fall back to defaults so the UI still renders instead of
        // getting stuck on a spinner forever (mirrors App's axTrusted
        // fallback for the same class of failure).
        console.error(err);
        setSettings(DEFAULT_SETTINGS);
      });
  }, []);

  const schedulePersist = useCallback((next: Settings) => {
    pending.current = next;
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    debounceTimer.current = setTimeout(() => {
      debounceTimer.current = null;
      const toSend = pending.current;
      pending.current = null;
      if (!toSend) return;
      persistChain.current = persistChain.current.then(() =>
        updateSettings(toSend),
      );
    }, PERSIST_DEBOUNCE_MS);
  }, []);

  // Flush any pending debounced write immediately (e.g. on unmount) so a
  // trailing edit within the debounce window isn't silently dropped.
  const flush = useCallback(() => {
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
      debounceTimer.current = null;
    }
    const toSend = pending.current;
    pending.current = null;
    if (toSend) {
      persistChain.current = persistChain.current.then(() =>
        updateSettings(toSend),
      );
    }
  }, []);

  useEffect(() => () => flush(), [flush]);

  const patch = useCallback(
    (updater: (s: Settings) => Settings) => {
      setSettings((cur) => {
        if (!cur) return cur;
        const next = updater(cur);
        schedulePersist(next);
        return next;
      });
    },
    [schedulePersist],
  );

  // Updates local state without persisting — for callers where the Go side
  // already saved the change (e.g. RestoreDefaultHotkeys) and a second
  // updateSettings would be redundant or could clobber other in-flight edits.
  const replaceLocal = useCallback((updater: (s: Settings) => Settings) => {
    setSettings((cur) => (cur ? updater(cur) : cur));
  }, []);

  return { settings, patch, replaceLocal };
}
