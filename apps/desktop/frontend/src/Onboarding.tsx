import { useEffect, useRef, useState } from "react";
import { axTrusted, requestAXPermission } from "./api";

type Props = { onGranted: () => void };

const SYSTEM_SETTINGS_URL =
  "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility";

// Shown instead of the tab shell until Accessibility access is granted.
// Polls axTrusted() every 2s so the UI flips to the shell as soon as the
// user grants access in System Settings — no restart required, since the
// Go side starts the engine as soon as AXTrusted() observes the grant.
export function Onboarding({ onGranted }: Props) {
  const [requesting, setRequesting] = useState(false);
  // macOS only shows the AXIsProcessTrustedWithOptions prompt once per
  // launch; a second click after the user dismisses/denies it does
  // nothing. Once we've asked, swap the button for a direct link into
  // System Settings instead of a dead "Grant Access…" button.
  const [promptedOnce, setPromptedOnce] = useState(false);

  // Keep the latest onGranted in a ref instead of the effect's deps so a
  // parent re-render (which often hands us a fresh inline closure) doesn't
  // tear down and restart the 2s poll interval.
  const onGrantedRef = useRef(onGranted);
  useEffect(() => {
    onGrantedRef.current = onGranted;
  }, [onGranted]);

  useEffect(() => {
    const id = setInterval(() => {
      axTrusted()
        .then((trusted) => {
          if (trusted) onGrantedRef.current();
        })
        .catch(() => {
          // Bindings unavailable (tests / plain-browser dev) — nothing to poll.
        });
    }, 2000);
    return () => clearInterval(id);
  }, []);

  const grant = async () => {
    setRequesting(true);
    try {
      await requestAXPermission();
    } finally {
      setRequesting(false);
      setPromptedOnce(true);
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
        {promptedOnce ? (
          <a
            href={SYSTEM_SETTINGS_URL}
            className="mt-6 block w-full rounded-lg bg-indigo-500 px-4 py-2 text-center text-sm font-medium text-white transition-colors hover:bg-indigo-400"
          >
            Open System Settings
          </a>
        ) : (
          <button
            type="button"
            onClick={grant}
            disabled={requesting}
            className="mt-6 w-full rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {requesting ? "Requesting…" : "Grant Access…"}
          </button>
        )}
        <p className="mt-4 text-xs text-zinc-500">
          System Settings → Privacy &amp; Security → Accessibility, then
          toggle Tiles Spliter on. Once granted, this window switches
          automatically.
        </p>
      </div>
    </div>
  );
}
