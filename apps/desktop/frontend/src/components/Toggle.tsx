type Props = {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
};

export function Toggle({ label, hint, checked, onChange }: Props) {
  return (
    <label className="flex items-start justify-between gap-4 py-3">
      <span>
        <span className="block text-sm font-medium">{label}</span>
        {hint && <span className="block text-xs text-zinc-400">{hint}</span>}
      </span>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-5 w-9 accent-indigo-500"
        role="switch"
        aria-checked={checked}
      />
    </label>
  );
}
