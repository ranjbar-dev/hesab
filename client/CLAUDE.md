# ./client — Company / individual dashboard

See root [`../CLAUDE.md`](../CLAUDE.md) for product, workflow, and always-on
tools. This file holds client-specific rules and learnings.

## Rules

- **Next.js 16.3.3**, App Router, Tailwind CSS v4.
- Persian, RTL, Jalali dates.
- **ui-ux-pro-max skill required** for every design/UI task here.
- Purpose: companies and individuals manage their accounting data.
- Talks to `./api` only — no direct DB access.
- Tenant-aware: user session scoped to one company.

## Learnings

<!-- LEARNINGS -->
- **2026-08-30 — Toasts for feedback.** `sonner` `<Toaster>` lives in `app/layout.tsx` (`theme="light"`, `richColors`, `dir="rtl"`, `position="top-center"`). Show all user-facing error / warning / info / success messages (login, future forgot/reset password, etc.) via `toast.error|warning|info|success("پیام فارسی")`, not inline message state. Keep new pages on the same pattern.
