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
