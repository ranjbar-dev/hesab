# ./api — Go backend

See root [`../CLAUDE.md`](../CLAUDE.md) for product, workflow, and always-on
tools. This file holds api-specific rules and learnings.

## Rules

- **Go + Gin** for HTTP routing/middleware.
- **sqlc** for all Postgres access. No ORM. Hand-written SQL under a `query/`
  dir, generated code committed.
- **JWT** for client authentication (admin and dashboard tokens separated).
- **DDD design pattern** — layered: `domain` (entities, rules) / `application`
  (use cases) / `infrastructure` (sqlc, external) / `interface` (Gin handlers).
- Multi-tenant: every company-scoped query filters by tenant/company id.
- Errors and API responses returned in a consistent envelope (shape decided at
  first task).

## Learnings

<!-- LEARNINGS -->
- **2026-08-30 — Admin auth responses.** Errors use `{"error":{"code":"...","message":"..."}}`; phones normalize to canonical `9XXXXXXXXX`.
- **2026-08-30 — Admin sessions.** Refresh tokens rotate and live in the HttpOnly `admin_refresh_token` cookie (`SameSite=None`; localhost is treated as secure for dev). Migrations are CLI-only golang-migrate files in `api/db/migration`; seed with `make seed`.
- **2026-08-30 — SMS.** Password-reset SMS is a fake fixed-code (`123456`) stub; sms.ir remains a TODO.
- **2026-08-30 — Rebuild the api container after backend changes.** `docker compose up -d` reuses the old image; a stale `hesab-api-1` silently serves pre-change code (no `/admin/*` routes, no CORS headers). Always `docker compose up -d --build api`. The container does not run migrations or the seeder — run `migrate ... up` and `go run ./cmd/seed` against the DB yourself.
- **2026-08-30 — Cross-origin auth, not a proxy.** The SPAs call the API directly at `http://localhost:8080`; `CORS()` middleware echoes allowed `Origin` + `Allow-Credentials`. `COOKIE_SECURE=true` is mandatory (SameSite=None needs Secure; localhost is a secure context so it still works on http). A same-origin `/api` rewrite was tried and abandoned — `output: "export"` makes Next ignore `rewrites` even in `next dev`.
