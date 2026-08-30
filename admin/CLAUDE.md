# ./admin — Super-admin panel

See root [`../CLAUDE.md`](../CLAUDE.md) for product, workflow, and always-on
tools. This file holds admin-specific rules and learnings.

## Rules

- **Next.js 16.3.3**, App Router, Tailwind CSS v4.
- Persian, RTL, Jalali dates.
- **ui-ux-pro-max skill required** for every design/UI task here.
- Purpose: manage companies, subscription plans, billing, activation and
  suspension.
- Talks to `./api` only — no direct DB access.

## Learnings

<!-- LEARNINGS -->
- **2026-08-30 — Admin auth.** Access tokens live in memory plus `sessionStorage`; refresh uses the HttpOnly cookie. `apiFetch` retries one refresh on authenticated 401s.
- **2026-08-30 — Auth pages.** Admin auth pages are client components and use `NEXT_PUBLIC_API_URL` for the API base URL (default `http://localhost:8080`, cross-origin).
- **2026-08-30 — No `/api` rewrite.** `output: "export"` makes `next.config` `rewrites` a no-op (Next warns and ignores them, even in `next dev`). The SPA talks to the API cross-origin and relies on the API's CORS middleware. A same-origin `/api` needs a real proxy (edge/CDN in prod) if ever wanted.
- **2026-08-30 — Toasts for feedback.** `sonner` `<Toaster>` lives in `app/layout.tsx` (`theme="dark"`, `richColors`, `dir="rtl"`, `position="top-center"`). Auth pages (`login`, `login/2fa`, `forgot-password`, `reset-password`, `settings/security`) show all validation / API errors and successes via `toast.error|success|info(...)`, not inline `<p>` message state. Keep new pages on the same pattern.
- **2026-08-30 — Dynamic routes under `output: "export"`.** `dynamicParams: true` is forbidden with static export; a `[id]` route needs `dynamicParams = false` + a `generateStaticParams` that returns at least one placeholder (we return `[{id:"_"}]`, which emits `out/users/_.html`). The `page.tsx` stays a server component doing only that; the real screen is a sibling `"use client"` component it renders (`app/users/[id]/Detail.tsx`, reads `useParams()`). Client-side nav (`router.push("/users/5")`) works; deep-link / hard-refresh needs `admin/serve.json` (`{"rewrites":[{"source":"/users/:id","destination":"/users/_.html"}]}`) — the `start` script passes it with `serve -c ../serve.json`. Verify the rewrite when a real static host is set up.
- **2026-08-30 — Jalali date input.** `jalaali-js` (~4 KB, zero deps) does the Jalali↔Gregorian conversion; `lib/jalali.ts` wraps it (`isoToJalaliLabel`, `jalaliDayRange`). The users-table date filter is three `<select>`s (year/month/day) that emit a UTC day range only once all three are set — no calendar-popover dependency. Persian digits via `toLocaleString("fa-IR")`; when parsing the current Jalali year force Latin digits (`nu-latn`) before `parseInt`.
- **2026-08-30 — Users admin screens.** `/users` (list + per-column debounced filter row + native `<dialog>` create modal + server pagination) and `/users/[id]` (profile, inline edit, enable/disable, reset-password, `confirm()`-gated soft-delete). Talks to `/admin/users*` (see `api/CLAUDE.md`). Client dashboard users are keyed by phone (immutable after create); one active account per phone.
