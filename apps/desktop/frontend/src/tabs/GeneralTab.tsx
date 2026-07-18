import type { Settings } from "@tiles-spliter/shared";

type Props = { settings: Settings; patch: (u: (s: Settings) => Settings) => void };

function Toggle({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-start justify-between gap-4 py-3">
      <span>
        <span className="block text-sm font-medium">{label}</span>
        {hint && <span className="block text-xs text-zinc-400">{hint}</span>}
      </span>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-5 w-9 accent-indigo-500"
        role="switch"
      />
    </label>
  );
}

export function GeneralTab({ settings, patch }: Props) {
  const g = settings.general;
  const set = (k: keyof Settings["general"], v: boolean | number) =>
    patch((s) => ({ ...s, general: { ...s.general, [k]: v } }));
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
          <span className="text-xs tabular-nums text-zinc-400">{g.windowPadding} pt</span>
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
