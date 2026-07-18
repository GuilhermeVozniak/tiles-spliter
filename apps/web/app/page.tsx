import { Hero } from "@/components/Hero";
import { HotkeyTable } from "@/components/HotkeyTable";

const RELEASES_URL =
  "https://github.com/GuilhermeVozniak/tiles-spliter/releases/latest";

const FEATURES = [
  {
    title: "Snap to Edges",
    description:
      "Drag a window to a screen edge or corner and it snaps into place — no fiddling with resize handles.",
  },
  {
    title: "Hotkeys for everything",
    description:
      "Every layout action has a default keyboard shortcut, fully remappable to fit your workflow.",
  },
  {
    title: "Multi-display",
    description:
      "Move windows between displays and cycle through them without ever touching the mouse.",
  },
];

export default function Home() {
  return (
    <>
      <Hero />

      <section className="mx-auto max-w-5xl px-6 py-16">
        <div className="grid gap-6 sm:grid-cols-3">
          {FEATURES.map((feature) => (
            <div
              key={feature.title}
              className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-6"
            >
              <h3 className="text-base font-semibold text-zinc-100">
                {feature.title}
              </h3>
              <p className="mt-2 text-sm text-zinc-400">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-5xl px-6 py-16">
        <h2 className="text-center text-2xl font-semibold text-zinc-100">
          Default hotkeys
        </h2>
        <p className="mx-auto mt-2 max-w-md text-center text-sm text-zinc-400">
          Sensible defaults out of the box — remap any of them in Settings.
        </p>
        <div className="mt-10">
          <HotkeyTable />
        </div>
      </section>

      <section className="mx-auto flex max-w-5xl flex-col items-center px-6 py-20">
        <h2 className="text-2xl font-semibold text-zinc-100">Ready to tile?</h2>
        <a
          href={RELEASES_URL}
          target="_blank"
          rel="noreferrer"
          className="mt-6 rounded-full bg-zinc-100 px-6 py-3 text-sm font-medium text-zinc-950 transition-colors hover:bg-white"
        >
          Download for macOS
        </a>
      </section>
    </>
  );
}
