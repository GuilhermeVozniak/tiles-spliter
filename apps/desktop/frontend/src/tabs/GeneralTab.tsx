import type { Settings } from "@tiles-spliter/shared";
import { Toggle } from "../components/Toggle";

type Props = {
  settings: Settings;
  patch: (u: (s: Settings) => Settings) => void;
  replaceLocal: (u: (s: Settings) => Settings) => void;
  cancelPending: () => void;
};

export function GeneralTab({ settings, patch }: Props) {
  const g = settings.general;
  const set = <K extends keyof Settings["general"]>(
    k: K,
    v: Settings["general"][K],
  ) => patch((s) => ({ ...s, general: { ...s.general, [k]: v } }));
  return (
    <div className="divide-y divide-zinc-800 px-6">
      <Toggle
        label="Launch Tiles Spliter at login"
        hint="Start automatically."
        checked={g.launchAtLogin}
        onChange={(v) => set("launchAtLogin", v)}
      />
      <Toggle
        label="Show in the menu bar"
        hint="When hidden, relaunch the app from Finder to open Preferences."
        checked={g.showMenuBarIcon}
        onChange={(v) => set("showMenuBarIcon", v)}
      />
      <Toggle
        label="Enable animations"
        checked={g.enableAnimations}
        onChange={(v) => set("enableAnimations", v)}
      />
      <div className="py-3">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">Windows padding</span>
          <span className="text-xs tabular-nums text-zinc-400">
            {g.windowPadding} pt
          </span>
        </div>
        <input
          type="range"
          min={0}
          max={50}
          value={g.windowPadding}
          onChange={(e) => set("windowPadding", Number(e.target.value))}
          className="mt-2 w-full accent-indigo-500"
        />
        <Toggle
          label="Enable for fullscreen windows"
          hint="Apply padding to fullscreen windows too."
          checked={g.padFullscreen}
          onChange={(v) => set("padFullscreen", v)}
        />
      </div>
    </div>
  );
}
