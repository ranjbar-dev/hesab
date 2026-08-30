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
- **2026-08-30 — App is business-scoped.** After login users land on
  `/select-business` (0 businesses → empty state + create CTA; exactly 1 & no
  invites → auto `router.replace` into it; else picker + pending-invites
  inbox). Real screens live under `/businesses/[id]/{dashboard,members,settings}`.
  The dynamic segment uses a **server** `businesses/[id]/layout.tsx` that
  exports `dynamicParams=false` + `generateStaticParams(){return[{id:"_"}]}`
  and renders a `"use client"` `Shell` (header, nav, business `<select>`
  switcher, membership guard, `BusinessContext` giving children
  `{business, role, reload}`). Sub-pages are plain `"use client"` — no per-page
  static-params. Static export emits `out/businesses/_/{dashboard,members,settings}.html`;
  `client/serve.json` rewrites `/businesses/:id[/seg]` to them and `start`
  passes `serve -c serve.json`. The chosen business is **only** in the URL id —
  never localStorage/cookie/context-persisted; the switcher just
  `router.push`es the same trailing segment under the new id.
  `lib/businesses.ts` has all fetchers + `roleLabels` + `canManage`.
  `owner`+`admin` see member controls + invite-by-phone (creates a pending
  invite); `accountant`/`viewer` get a read-only member list. Only `owner`
  soft-deletes; non-owners get «خروج از کسب‌وکار» (removes own membership).
