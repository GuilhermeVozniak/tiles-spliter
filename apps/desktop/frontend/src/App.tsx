import { useState } from "react";
import { AboutTab } from "./tabs/AboutTab";
import { GeneralTab } from "./tabs/GeneralTab";
import { HotkeysTab } from "./tabs/HotkeysTab";
import { SnapTab } from "./tabs/SnapTab";
import { useSettings } from "./useSettings";

const TABS = [
  { id: "general", label: "General", icon: "⚙", Component: GeneralTab },
  { id: "hotkeys", label: "Hotkeys", icon: "⌘", Component: HotkeysTab },
  { id: "snap", label: "Snap to Edges", icon: "◰", Component: SnapTab },
  { id: "about", label: "About", icon: "ⓘ", Component: AboutTab },
] as const;

type TabID = (typeof TABS)[number]["id"];

function Spinner() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div
        className="h-6 w-6 animate-spin rounded-full border-2 border-zinc-700 border-t-indigo-500"
        role="status"
        aria-label="Loading"
      />
    </div>
  );
}

export function App() {
  const { settings, patch } = useSettings();
  const [active, setActive] = useState<TabID>("general");

  const activeTab = TABS.find((t) => t.id === active) ?? TABS[0];
  const ActiveComponent = activeTab.Component;

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-zinc-950 text-zinc-100">
      <nav className="flex w-16 flex-col items-center gap-1 border-r border-zinc-800 bg-zinc-900/60 py-4">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            title={tab.label}
            onClick={() => setActive(tab.id)}
            className={`flex h-12 w-12 flex-col items-center justify-center gap-0.5 rounded-lg text-lg transition-colors ${
              tab.id === active
                ? "bg-indigo-500/20 text-indigo-300"
                : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200"
            }`}
          >
            <span aria-hidden="true">{tab.icon}</span>
            <span className="text-[9px] font-medium leading-none">{tab.label.split(" ")[0]}</span>
          </button>
        ))}
      </nav>
      <main className="flex-1 overflow-y-auto py-6">
        {settings ? <ActiveComponent settings={settings} patch={patch} /> : <Spinner />}
      </main>
    </div>
  );
}
