import { useEffect, useState } from "react";
import { axTrusted, requestAXPermission } from "./api";

type Props = { onGranted: () => void };

// Shown instead of the tab shell until Accessibility access is granted.
// Polls axTrusted() every 2s so the UI flips to the shell as soon as the
// user grants access in System Settings — no restart required, since the
// Go side starts the engine as soon as AXTrusted() observes the grant.
export function Onboarding({ onGranted }: Props) {
  const [requesting, setRequesting] = useState(false);

  useEffect(() => {
    const id = setInterval(() => {
      axTrusted()
        .then((trusted) => {
          if (trusted) onGranted();
        })
        .catch(() => {
          // Bindings unavailable (tests / plain-browser dev) — nothing to poll.
        });
    }, 2000);
    return () => clearInterval(id);
  }, [onGranted]);

  const grant = async () => {
    setRequesting(true);
    try {
      await requestAXPermission();
    } finally {
      setRequesting(false);
    }
  };

  return (
    <div className="flex h-screen w-screen items-center justify-center bg-zinc-950 text-zinc-100">
      <div className="mx-6 w-full max-w-sm rounded-2xl border border-zinc-800 bg-zinc-900/60 p-8 text-center">
        <div
          aria-hidden="true"
          className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-xl bg-indigo-500/20 text-3xl text-indigo-300"
        >
          ◰
        </div>
        <h1 className="text-lg font-semibold">Tiles Spliter</h1>
        <p className="mt-2 text-sm text-zinc-400">
          Tiles Spliter needs Accessibility access to move and resize windows.
        </p>
        <button
          type="button"
          onClick={grant}
          disabled={requesting}
          className="mt-6 w-full rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {requesting ? "Requesting…" : "Grant Access…"}
        </button>
        <p className="mt-4 text-xs text-zinc-500">
          Opens System Settings → Privacy &amp; Security → Accessibility. Once
          granted, this window switches automatically.
        </p>
      </div>
    </div>
  );
}
