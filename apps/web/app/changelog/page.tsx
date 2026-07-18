import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Changelog — Tiles Spliter",
};

const RELEASES = [
  {
    version: "0.1.2",
    note: "Styled DMG installer with drag-to-Applications window.",
  },
  {
    version: "0.1.1",
    note: "Menu bar parity: tile icons, section separators and live shortcut hints in the tray menu. Plus drag, hotkey and settings reliability fixes.",
  },
  { version: "0.1.0", note: "Initial release." },
];

export default function ChangelogPage() {
  return (
    <section className="mx-auto max-w-2xl px-6 py-20">
      <h1 className="text-3xl font-semibold text-zinc-50">Changelog</h1>
      <ul className="mt-8 space-y-4">
        {RELEASES.map((release) => (
          <li key={release.version} className="border-b border-zinc-800 pb-4">
            <span className="font-mono text-sm text-zinc-400">
              {release.version}
            </span>
            <span className="mx-2 text-zinc-600">—</span>
            <span className="text-sm text-zinc-200">{release.note}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}
