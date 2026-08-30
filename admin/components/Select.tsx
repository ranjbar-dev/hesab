"use client";

import RSelect, { type Props as RSelectProps } from "react-select";

export type Option = { value: string; label: string };

export default function Select({ placeholder, ...props }: RSelectProps<Option, false> & { placeholder?: string }) {
  return <RSelect {...props} isRtl placeholder={placeholder ?? "انتخاب کنید"} noOptionsMessage={() => "موردی یافت نشد"} classNamePrefix="rs" unstyled classNames={{
    control: (s) => `min-h-11 rounded-lg border px-1 text-sm ${s.isFocused ? "border-brand-accent ring-2 ring-brand-accent/30" : "border-brand-border"} bg-brand-bg`,
    menu: () => "mt-1 overflow-hidden rounded-lg border border-brand-border bg-brand-surface text-sm text-brand-text shadow-lg",
    option: (s) => `cursor-pointer px-3 py-2 ${s.isFocused ? "bg-brand-accent/10" : ""} ${s.isSelected ? "bg-brand-accent text-white" : ""}`,
    placeholder: () => "text-brand-muted", singleValue: () => "text-brand-text", input: () => "text-brand-text", indicatorSeparator: () => "hidden", dropdownIndicator: () => "px-2 text-brand-muted",
  }} />;
}
