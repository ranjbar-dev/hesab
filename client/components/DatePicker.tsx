"use client";
import DP from "react-multi-date-picker";
import persian from "react-date-object/calendars/persian";
import persian_fa from "react-date-object/locales/persian_fa";
type Range = { from: string; to: string };
const boundary = (date: Date) => {
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
  onChange: (value: string | null) => void;
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
      inputClass="h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm text-brand-text"
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
  onChange: (value: Range | null) => void;
  placeholder?: string;
}) {
  return (
    <DP
      calendar={persian}
      locale={persian_fa}
      range
      value={value ? [new Date(value.from), new Date(value.to)] : []}
      onChange={(v) => {
        const dates = v as unknown as Array<{ toDate(): Date }>;
        if (!dates?.length) return onChange(null);
        if (dates.length < 2) return;
        const a = boundary(dates[0].toDate()),
          b = boundary(dates[1].toDate());
        onChange({ from: a.from, to: b.to });
      }}
      inputClass="h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm text-brand-text"
      placeholder={placeholder ?? "بازه تاریخ"}
      calendarPosition="bottom-right"
    />
  );
}
