import type { Settings } from "@tiles-spliter/shared";

type Props = {
  settings: Settings;
  patch: (u: (s: Settings) => Settings) => void;
  replaceLocal: (u: (s: Settings) => Settings) => void;
  cancelPending: () => void;
};

// Single source of truth for the displayed version. Kept as a local constant
// (rather than importing package.json's own "version" field) so this file
// stays self-contained — bump alongside apps/desktop/build/Info.plist.
const VERSION = "0.1.3";
const REPO_URL = "https://github.com/GuilhermeVozniak/tiles-spliter";

export function AboutTab(_props: Props) {
  return (
    <div className="flex h-full flex-col items-center justify-center px-6 text-center">
      <div
        aria-hidden="true"
        className="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-indigo-500/20 text-3xl text-indigo-300"
      >
        ◰
      </div>
      <h1 className="text-lg font-semibold">Tiles Spliter</h1>
      <p className="mt-1 text-xs tabular-nums text-zinc-500">
        Version {VERSION}
      </p>
      <p className="mt-4 max-w-xs text-sm text-zinc-400">
        Snap, tile and arrange your windows — by drag, hotkey or menu bar.
      </p>
      <a
        href={REPO_URL}
        target="_blank"
        rel="noreferrer"
        className="mt-4 text-sm text-indigo-300 transition-colors hover:text-indigo-200"
      >
        Website &amp; releases
      </a>
      <p className="mt-8 max-w-xs text-xs text-zinc-500">
        Snap-to-edges and hotkey techniques inspired by{" "}
        <a
          href="https://rectangleapp.com"
          target="_blank"
          rel="noreferrer"
          className="underline decoration-zinc-700 hover:text-zinc-300"
        >
          Rectangle
        </a>
        , the MIT-licensed macOS window manager that came before it.
      </p>
    </div>
  );
}
