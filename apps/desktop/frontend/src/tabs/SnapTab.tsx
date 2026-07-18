import type { Action, Settings, ZoneID } from "@tiles-spliter/shared";
import { ACTION_LABELS, ACTIONS, ZONES } from "@tiles-spliter/shared";

type Props = {
  settings: Settings;
  patch: (u: (s: Settings) => Settings) => void;
};

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
        aria-checked={checked}
      />
    </label>
  );
}

const ZONE_POS: Record<ZoneID, string> = {
  "top-left": "left-2 top-2",
  top: "left-1/2 top-2 -translate-x-1/2",
  "top-right": "right-2 top-2",
  left: "left-2 top-1/2 -translate-y-1/2",
  right: "right-2 top-1/2 -translate-y-1/2",
  "bottom-left": "left-2 bottom-2",
  bottom: "left-1/2 bottom-2 -translate-x-1/2",
  "bottom-right": "right-2 bottom-2",
};

export function SnapTab({ settings, patch }: Props) {
  const { snap } = settings;

  const setSnap = <K extends keyof Settings["snap"]>(
    k: K,
    v: Settings["snap"][K],
  ) => patch((s) => ({ ...s, snap: { ...s.snap, [k]: v } }));

  const setZone = (zone: ZoneID, action: Action) =>
    patch((s) => ({
      ...s,
      snap: { ...s.snap, zones: { ...s.snap.zones, [zone]: action } },
    }));

  return (
    <div className="px-6">
      <div className="divide-y divide-zinc-800">
        <Toggle
          label="Snap windows when moved to the edges of the screen"
          checked={snap.enabled}
          onChange={(v) => setSnap("enabled", v)}
        />
        <Toggle
          label="Restore previous window size when moved again"
          checked={snap.restorePreviousSize}
          onChange={(v) => setSnap("restorePreviousSize", v)}
        />
      </div>

      <div className={snap.enabled ? "" : "pointer-events-none opacity-40"}>
        <div className="mt-4 rounded-xl border border-zinc-800 bg-zinc-900/40 p-4">
          <span className="block text-sm font-medium">Active Edges</span>
          <div className="mt-3 flex justify-center">
            <div className="relative aspect-[16/10] w-full max-w-md rounded-lg bg-gradient-to-br from-zinc-800 to-zinc-900 shadow-inner">
              {ZONES.map((zone) => (
                <select
                  key={zone}
                  value={snap.zones[zone]}
                  onChange={(e) => setZone(zone, e.target.value as Action)}
                  className={`absolute ${ZONE_POS[zone]} rounded-md bg-zinc-800/90 px-2 py-1 text-xs text-zinc-100`}
                >
                  {ACTIONS.map((a) => (
                    <option key={a} value={a}>
                      {ACTION_LABELS[a]}
                    </option>
                  ))}
                </select>
              ))}
            </div>
          </div>
          <p className="mt-3 text-center text-xs text-zinc-400">
            Hold ⌥ ALT or ⌘ CMD while dragging a window to the left or right
            edge to snap it to thirds.
          </p>
        </div>

        <div className="py-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Snap zone thickness</span>
            <span className="text-xs tabular-nums text-zinc-400">
              {snap.zoneThickness} pt
            </span>
          </div>
          <input
            type="range"
            min={1}
            max={40}
            value={snap.zoneThickness}
            onChange={(e) => setSnap("zoneThickness", Number(e.target.value))}
            className="mt-2 w-full accent-indigo-500"
          />
        </div>

        <div className="py-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Delay before activation</span>
            <span className="text-xs tabular-nums text-zinc-400">
              {snap.activationDelayMs === 0
                ? "None"
                : `${snap.activationDelayMs} ms`}
            </span>
          </div>
          <input
            type="range"
            min={0}
            max={1000}
            step={50}
            value={snap.activationDelayMs}
            onChange={(e) =>
              setSnap("activationDelayMs", Number(e.target.value))
            }
            className="mt-2 w-full accent-indigo-500"
          />
        </div>
      </div>
    </div>
  );
}
