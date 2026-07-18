import { downloadUrl } from "@tiles-spliter/shared";

export function Hero() {
  return (
    <section className="mx-auto flex max-w-5xl flex-col items-center px-6 pb-16 pt-20 text-center">
      <h1 className="text-4xl font-semibold tracking-tight text-zinc-50 sm:text-6xl">
        Tiles Spliter
      </h1>
      <p className="mt-5 max-w-xl text-balance text-lg text-zinc-400">
        Snap, tile and arrange your Mac windows — by drag, hotkey or menu bar.
      </p>
      <div className="mt-8 flex flex-col items-center gap-3">
        <a
          href={downloadUrl()}
          target="_blank"
          rel="noreferrer"
          className="rounded-full bg-zinc-100 px-6 py-3 text-sm font-medium text-zinc-950 transition-colors hover:bg-white"
        >
          Download for macOS
        </a>
        <p className="text-xs text-zinc-500">Requires macOS 13+</p>
      </div>

      {/* Pure-CSS animated tiling demo */}
      <div className="relative mt-16 h-72 w-full max-w-xl overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/60 shadow-2xl">
        <div className="absolute inset-x-0 top-0 flex h-6 items-center gap-1.5 border-b border-zinc-800 bg-zinc-900 px-3">
          <span className="h-2.5 w-2.5 rounded-full bg-red-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-yellow-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-green-500/70" />
        </div>
        <div className="tile-demo-window-left absolute rounded-md border border-blue-400/40 bg-blue-500/20 backdrop-blur-sm" />
        <div className="tile-demo-window-right absolute rounded-md border border-violet-400/40 bg-violet-500/20 backdrop-blur-sm" />
      </div>
    </section>
  );
}
