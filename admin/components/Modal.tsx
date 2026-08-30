"use client";
import { useEffect, useRef, type ReactNode } from "react";
export default function Modal({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      onClick={(e) => {
        if (e.target === ref.current) onClose();
      }}
      className="m-auto max-h-[85vh] w-full max-w-md overflow-y-auto rounded-2xl border border-brand-border bg-brand-surface p-6 text-brand-text backdrop:bg-black/50"
    >
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-bold">{title}</h2>
        <button
          type="button"
          aria-label="بستن"
          onClick={onClose}
          className="cursor-pointer rounded-lg p-1 text-brand-muted transition-colors hover:bg-brand-bg focus-visible:ring-2 focus-visible:ring-brand-accent/30"
        >
          <svg
            viewBox="0 0 24 24"
            className="h-5 w-5"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <path d="M18 6 6 18M6 6l12 12" />
          </svg>
        </button>
      </div>
      {children}
    </dialog>
  );
}
