import type { Settings } from "@tiles-spliter/shared";

type Props = { settings: Settings; patch: (u: (s: Settings) => Settings) => void };

// Stub — replaced by Task 19.
export function SnapTab(_props: Props) {
  return (
    <div className="flex h-full items-center justify-center px-6 text-center">
      <p className="text-sm text-zinc-400">Coming in a later task</p>
    </div>
  );
}
