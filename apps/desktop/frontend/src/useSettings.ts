import type { Settings } from "@tiles-spliter/shared";
import { useCallback, useEffect, useState } from "react";
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
