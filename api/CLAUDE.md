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
- **2026-08-30 — Client authentication.** Client auth deliberately parallels `adminauth` while staying separate: it uses `users`, `/client/auth/*` routes, and the `client_refresh_token` HttpOnly cookie scoped to `/client/auth`; the seed user is `9120000000` / `Client@Pass1999`.
- **2026-08-30 — Admin-side user management.** `/admin/users*` (under the existing `AdminAuth` group) is `application/usersadmin` + `repo.UserAdminRepo` + `query/users_admin.sql` (`Admin*`-prefixed to avoid clashing with `clientauth`'s queries on the same `users` table). Migration `000003` added `national_id` (nullable, **not** unique — only `phone_number` is), `account_type` (`individual`/`company`), `status` (`active`/`disabled`), `deleted_at`; it dropped `users_email_key` and defaults `email` to `''` (blank stays `''`, never `NULL`, so `sqlc.User.Email` stays `string` and `clientauth` keeps compiling — new columns are additive for the same reason). Disable, reset-password and soft-delete all call `RevokeAllUserRefreshTokens`. List filtering uses `sqlc.narg` optional predicates + `ILIKE` substring; pagination is `lim`/`off` from clamped `page`/`page_size`. Duplicate phone → pg `23505` → `user.ErrPhoneTaken` → HTTP 409. Soft-deleted phone stays reserved (hard `UNIQUE`).
- **2026-08-30 — Admin auth responses.** Errors use `{"error":{"code":"...","message":"..."}}`; phones normalize to canonical `9XXXXXXXXX`.
- **2026-08-30 — Admin sessions.** Refresh tokens rotate and live in the HttpOnly `admin_refresh_token` cookie (`SameSite=None`; localhost is treated as secure for dev). Migrations are CLI-only golang-migrate files in `api/db/migration`; seed with `go run ./cmd/seed`.
- **2026-08-30 — SMS.** Password-reset SMS is a fake fixed-code (`123456`) stub; sms.ir remains a TODO.
- **2026-08-30 — Rebuild the api container after backend changes.** `docker compose up -d` reuses the old image; a stale `hesab-api-1` silently serves pre-change code (no `/admin/*` routes, no CORS headers). Always `docker compose up -d --build api`. The container does not run migrations or the seeder — run `migrate ... up` and `go run ./cmd/seed` against the DB yourself.
- **2026-08-30 — Cross-origin auth, not a proxy.** The SPAs call the API directly at `http://localhost:8080`; `CORS()` middleware echoes allowed `Origin` + `Allow-Credentials`. `COOKIE_SECURE=true` is mandatory (SameSite=None needs Secure; localhost is a secure context so it still works on http). A same-origin `/api` rewrite was tried and abandoned — `output: "export"` makes Next ignore `rewrites` even in `next dev`.
- **2026-08-30 — No `make` on this Windows box.** `api/Makefile` exists but the `make` binary is not installed. Run the targets directly: `migrate -path db/migration -database "$DATABASE_URL" up`, `go run ./cmd/seed`, `go run ./cmd/server`. The golang-migrate CLI is at `~/go/bin/migrate`.
- **2026-08-30 — gopls lags after new struct fields.** After adding fields to `config.Config`, the language server kept reporting `undefined: config.Config.<field>` while `go build ./...` was clean. `go build` / `go test` are the source of truth, not IDE diagnostics (same rule as the Next IDE-lag note in the root `CLAUDE.md`).
- **2026-08-30 — Auth stack + flow shapes.** JWT via `github.com/golang-jwt/jwt/v5` (HS256); TOTP 2FA via `github.com/pquerna/otp` (`totp.Generate` / `totp.Validate`, Google-Authenticator defaults). Patterns to reuse when client auth is built: login is two-step when 2FA is on (password → short-lived `pending_token` → TOTP code → tokens); password reset is 3-step OTP (`forgot` → SMS code → `reset` with phone + code + new password); the refresh token is an opaque 32-byte random stored as a SHA-256 hash, rotated on every refresh, revoked on logout and on password reset.
