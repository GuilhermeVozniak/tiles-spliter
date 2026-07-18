import {
  actionLabel,
  DEFAULT_SETTINGS,
  HOTKEY_DISPLAY,
} from "@tiles-spliter/shared";

export function HotkeyTable() {
  const entries = Object.entries(DEFAULT_SETTINGS.hotkeys.bindings);
  return (
    <table className="mx-auto text-sm">
      <tbody>
        {entries.map(([action, hk]) => (
          <tr key={action} className="border-b border-zinc-800">
            <td className="py-2 pr-8">{actionLabel(action)}</td>
            <td className="py-2 font-mono text-zinc-400">
              {HOTKEY_DISPLAY(hk)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
