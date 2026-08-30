"use client";
export default function FormField({
  label,
  value,
  onChange,
  type = "text",
  required = true,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  required?: boolean;
  placeholder?: string;
}) {
  const id = `field-${label}`;
  return (
    <label htmlFor={id} className="block text-sm">
      {label}
      <input
        id={id}
        required={required}
        type={type}
        inputMode={type === "tel" ? "numeric" : undefined}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? label}
        className="mt-2 h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm focus-visible:ring-2 focus-visible:ring-brand-accent/30"
      />
    </label>
  );
}
