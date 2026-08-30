"use client";

import DP from "react-multi-date-picker";
import persian from "react-date-object/calendars/persian";
import persian_fa from "react-date-object/locales/persian_fa";

type Range = { from: string; to: string };
const dayRange = (date: Date): Range => {
  const from = new Date(
    Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()),
  );
  const to = new Date(from);
  to.setUTCDate(to.getUTCDate() + 1);
  to.setUTCMilliseconds(-1);
  return { from: from.toISOString(), to: to.toISOString() };
};
export function DatePicker({
  value,
  onChange,
  placeholder,
}: {
  value: string | null;
  onChange: (iso: string | null) => void;
  placeholder?: string;
}) {
  return (
    <DP
      calendar={persian}
      locale={persian_fa}
      value={value ? new Date(value) : null}
      onChange={(d) =>
        onChange(d ? (d as { toDate(): Date }).toDate().toISOString() : null)
      }
      inputClass="h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm text-brand-text focus-visible:ring-2 focus-visible:ring-brand-accent/30"
      placeholder={placeholder ?? "انتخاب تاریخ"}
      calendarPosition="bottom-right"
    />
  );
}
export function DateRangePicker({
  value,
  onChange,
  placeholder,
}: {
  value: Range | null;
  onChange: (v: Range | null) => void;
  placeholder?: string;
}) {
  return (
    <DP
      calendar={persian}
      locale={persian_fa}
      range
      value={value ? [new Date(value.from), new Date(value.to)] : []}
      onChange={(dates) => {
        const range = dates as unknown as Array<{ toDate(): Date }>;
        if (!range?.length) return onChange(null);
        if (range.length < 2) return;
        const first = dayRange(range[0].toDate()),
          last = dayRange(range[1].toDate());
        onChange({ from: first.from, to: last.to });
      }}
      inputClass="h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm text-brand-text focus-visible:ring-2 focus-visible:ring-brand-accent/30"
      placeholder={placeholder ?? "بازه تاریخ"}
      calendarPosition="bottom-right"
    />
  );
}
