import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Privacy — Tiles Spliter",
};

export default function PrivacyPage() {
  return (
    <section className="mx-auto max-w-2xl px-6 py-20">
      <h1 className="text-3xl font-semibold text-zinc-50">Privacy</h1>
      <p className="mt-6 text-base leading-relaxed text-zinc-300">
        Tiles Spliter runs entirely on your Mac. No analytics, no network calls,
        no data collection. Accessibility permission is used only to move and
        resize windows.
      </p>
    </section>
  );
}
