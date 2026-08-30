# Plan — Admin Authentication (phone + password, refresh tokens, optional TOTP 2FA)

Orchestrator: Claude. Coder: **you (Codex)**. Follow this plan exactly. Use the
**context7 MCP server** for current docs on every library you touch
(golang-migrate, golang-jwt/jwt v5, pquerna/otp, gin, sqlc, Next.js 16).
Ponytail rules apply: smallest diff that works, stdlib first, no speculative
abstractions. Caveman prose is for chat only — write normal code + comments.

## Scope

- **Admin only.** Client auth is a later task (it will reuse this phone-based
  pattern, so keep names admin-scoped, e.g. `/admin/auth/...`).
- Auth identifier is **phone number, never email**.
- Backend (Go/Gin/sqlc) + admin frontend (Next.js static-export SPA) + DB
  migrations + a seeder command.

## Decisions (already made — do not re-litigate)

| Topic | Decision |
|---|---|
| Reset flow | 3-step OTP: `forgot-password` sends 6-digit code via SMS; `reset-password {phone, code, new_password}` verifies + sets password. OTP hashed in DB, 5-min TTL, single use. |
| 2FA | RFC 6238 **TOTP** (authenticator app), library `github.com/pquerna/otp`. Two-step login: password check → if 2FA on, return short-lived `pending_token` (no access); second call `{pending_token, code}` returns real tokens. Not required; admin opts in. |
| Refresh token | Opaque 32-byte random (NOT a JWT), stored **hashed** in `admin_refresh_tokens`. Rotated on every `/refresh` (old row revoked, new issued). `/logout` revokes. Delivered as **HttpOnly cookie**; access token (JWT) in JSON body. |
| Access token | JWT HS256, 15-min TTL, claims `sub`=admin id, `typ`="admin", `iat`, `exp`. Secret from `JWT_SECRET`. |
| Migrations | **golang-migrate** versioned files under `api/db/migration` (CLI already installed at `~/go/bin/migrate`; sqlc.yaml already points `schema` there). |
| Seeder | `api/cmd/seed/main.go`, idempotent upsert by phone. |
| Phone format | Normalize to bare 10-digit `9XXXXXXXXX`: strip `+98`, `0098`, leading `0`, spaces, dashes. Validate `^9\d{9}$`. Same normalization on every write and every lookup. |
| 2FA UI | Backend endpoints **and** an admin-panel security settings page. |
| SMS | Interface + fake impl that logs and "delivers" the fixed code `123456`. `TODO` comment: implement sms.ir provider. |

## Seed admin

phone `9370843199` → normalized `9370843199`; password `Amir@Pass1999`
(bcrypt); `first_name` `Amir`, `last_name` `Admin`, `email`
`admin@hesab.local`, `is_male` `true`, `totp_secret` `''` (2FA off).
Upsert: `ON CONFLICT (phone_number) DO UPDATE` password + names + email so
re-running the seeder is safe.

---

## 1. Database — `api/db/migration/000001_admin_auth.{up,down}.sql`

Create with: `migrate create -ext sql -dir api/db/migration -seq admin_auth`
(gives `000001_admin_auth.up.sql` / `.down.sql`).

### up

```sql
CREATE TABLE admins (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_name    TEXT        NOT NULL,
    last_name     TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    phone_number  TEXT        NOT NULL UNIQUE,            -- canonical 9XXXXXXXXX
    is_male       BOOLEAN     NOT NULL,
    password_hash TEXT        NOT NULL,
    totp_secret   TEXT        NOT NULL DEFAULT '',        -- '' = 2FA disabled
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE admin_refresh_tokens (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    admin_id   BIGINT      NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,               -- sha256 hex
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX admin_refresh_tokens_admin_id_idx ON admin_refresh_tokens (admin_id);

CREATE TABLE admin_password_resets (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    admin_id    BIGINT      NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    code_hash   TEXT        NOT NULL,                     -- sha256 hex of 6-digit code
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX admin_password_resets_admin_id_idx ON admin_password_resets (admin_id);
```

`down`: `DROP TABLE` all three (reverse order).

2FA enabled state is derived: `totp_secret <> ''`. No separate boolean.

## 2. sqlc queries — `api/query/admin_auth.sql`

Name queries for sqlc (`-- name: X :one|:many|:exec`). Needed:

- `GetAdminByPhone` :one — by `phone_number`.
- `GetAdminByID` :one.
- `CreateAdmin` :one — insert, return row (used by seeder path if you prefer SQL over raw).
- `UpdateAdminPassword` :exec — set `password_hash` by id.
- `SetAdminTOTPSecret` :exec — set `totp_secret` by id (activate = secret, disable = '').
- `InsertRefreshToken` :one — (`admin_id`, `token_hash`, `expires_at`).
- `GetRefreshToken` :one — by `token_hash` (return row incl. `revoked_at`, `expires_at`, `admin_id`).
- `RevokeRefreshToken` :exec — set `revoked_at = now()` by `token_hash` where `revoked_at IS NULL`.
- `RevokeAllAdminRefreshTokens` :exec — by `admin_id` (used on password reset).
- `InsertPasswordReset` :one — (`admin_id`, `code_hash`, `expires_at`).
- `InvalidateAdminPasswordResets` :exec — set `consumed_at = now()` for all unconsumed rows of an admin (call before inserting a new one).
- `GetLatestPasswordReset` :one — newest row for `admin_id` where `consumed_at IS NULL AND expires_at > now()`.
- `ConsumePasswordReset` :exec — set `consumed_at = now()` by id.

Run `sqlc generate` (config `api/sqlc.yaml`, out `internal/infrastructure/db/sqlc`). Commit generated code.

## 3. Go dependencies

`cd api && go get`:
- `github.com/golang-jwt/jwt/v5`
- `github.com/pquerna/otp`
- `golang.org/x/crypto/bcrypt` is already in the module graph (indirect) — it becomes direct.

Do **not** add golang-migrate to `go.mod` (CLI-only). Run `go mod tidy`.
Local Go is 1.24.2 but `go.mod` says `go 1.25` (documented, keep it) — the Go
toolchain auto-downloads 1.25; do not touch `go.mod`/`Dockerfile` versions.

## 4. Config — `api/internal/config/config.go`

Add fields + env (keep the `getenv` helper style; add `getenvInt`/`getdur` as
needed):

| Field | Env | Dev default |
|---|---|---|
| `JWTSecret` | `JWT_SECRET` | `dev-insecure-admin-secret-change-me` |
| `AccessTokenTTL` | `ACCESS_TOKEN_TTL` | `15m` |
| `RefreshTokenTTL` | `REFRESH_TOKEN_TTL` | `720h` |
| `PasswordResetTTL` | `PASSWORD_RESET_TTL` | `5m` |
| `TwoFAPendingTTL` | `TWOFA_PENDING_TTL` | `5m` |
| `CORSOrigins` | `CORS_ORIGINS` (comma-sep) | `http://localhost:3010,http://localhost:3020` |
| `CookieSecure` | `COOKIE_SECURE` | `true` |
| `CookieDomain` | `COOKIE_DOMAIN` | `` (empty) |
| `TOTPIssuer` | `TOTP_ISSUER` | `Hesab Admin` |

Update root `.env.example` and the `api` service `environment:` block in
`docker-compose.yml` with the new vars (dev values).

## 5. Domain — `api/internal/domain/admin/`

- `admin.go`: `Admin` struct (id, first/last name, email, phone, isMale,
  passwordHash, totpSecret, createdAt). Method `TwoFAEnabled() bool` =
  `a.TOTPSecret != ""`. Sentinel errors: `ErrInvalidCredentials`,
  `ErrResetCodeInvalid`, `ErrRefreshInvalid`, `ErrTwoFARequired`,
  `ErrTwoFACodeInvalid`, `ErrWeakPassword`.
- `phone.go`: `NormalizePhone(raw string) (string, error)` — trim, remove
  spaces/`-`/`(`/`)`, map `+98`/`0098`/leading `00`/leading `0` prefixes to
  bare, then require `^9\d{9}$`. Reject otherwise.
- `password.go`: `ValidatePassword(string) error` — min 8 chars, at least one
  letter and one digit. ponytail: keep this small; do not add a zxcvbn dep.

## 6. Application — `api/internal/application/adminauth/service.go`

`Service` struct with injected interfaces (define them **here**, implement in
infrastructure):

```go
type Repository interface {
    AdminByPhone(ctx, phone string) (admin.Admin, error)   // wraps sql.ErrNoRows -> ErrInvalidCredentials at call site
    AdminByID(ctx, id int64) (admin.Admin, error)
    UpdatePassword(ctx, id int64, hash string) error
    SetTOTPSecret(ctx, id int64, secret string) error
    InsertRefreshToken(ctx, adminID int64, hash string, exp time.Time) error
    RefreshTokenByHash(ctx, hash string) (RefreshToken, error)
    RevokeRefreshToken(ctx, hash string) error
    RevokeAllRefreshTokens(ctx, adminID int64) error
    InvalidatePasswordResets(ctx, adminID int64) error
    InsertPasswordReset(ctx, adminID int64, codeHash string, exp time.Time) error
    LatestPasswordReset(ctx, adminID int64) (PasswordReset, error)
    ConsumePasswordReset(ctx, id int64) error
}
type TokenIssuer interface {           // implemented by infrastructure/token
    IssueAccess(adminID int64) (token string, expiresIn int, err error)
    IssuePending(adminID int64) (token string, err error)
    ParseAccess(token string) (adminID int64, err error)
    ParsePending(token string) (adminID int64, err error)
}
type SMSSender interface { Send(ctx, phone, message string) error }
type CodeGenerator func() string        // fake returns "123456"; real = random 6-digit
type Clock func() time.Time
```

Use cases (methods):

- `Login(ctx, phone, password) (LoginResult, error)` — normalize phone, fetch
  admin, `bcrypt.CompareHashAndPassword`; constant-ish failure (still run a
  dummy bcrypt compare when admin not found to reduce timing signal —
  ponytail-acceptable one-liner). If `TwoFAEnabled` → `LoginResult{TwoFARequired:true, PendingToken: IssuePending(id)}`. Else
  issue access + create refresh (return raw refresh string for the handler to
  set as cookie).
- `LoginVerify2FA(ctx, pendingToken, code) (Tokens, error)` — parse pending,
  load admin, `totp.Validate(code, admin.TOTPSecret)`; on ok issue access +
  refresh.
- `Refresh(ctx, rawRefresh) (Tokens, error)` — hash, look up, check not
  revoked / not expired; revoke old, insert new, issue access. Return new raw
  refresh.
- `Logout(ctx, rawRefresh) error` — hash + revoke; nil even if not found
  (idempotent).
- `ForgotPassword(ctx, phone) error` — normalize; look up admin; if missing,
  return nil (no enumeration). Else invalidate old resets, generate code via
  `CodeGenerator`, store sha256(code), TTL, then
  `SMSSender.Send(phone, "<code> کد بازیابی رمز عبور شماست")`.
- `ResetPassword(ctx, phone, code, newPassword) error` — normalize; validate
  new password; load admin; `LatestPasswordReset`; compare sha256(code);
  on ok: consume it, update password hash, `RevokeAllRefreshTokens`.
- `Setup2FA(ctx, adminID) (secret, otpauthURL string, err error)` —
  `totp.Generate({Issuer: cfg.TOTPIssuer, AccountName: admin.PhoneNumber})`;
  return `key.Secret()`, `key.URL()`. **Not persisted.**
- `Activate2FA(ctx, adminID, secret, code) error` — `totp.Validate(code, secret)`;
  on ok `SetTOTPSecret(id, secret)`.
- `Disable2FA(ctx, adminID, password) error` — verify password; `SetTOTPSecret(id, "")`.
- `Me(ctx, adminID) (admin.Admin, error)`.

Hashing helper: `sha256hex(s string) string` in a small `internal/pkg/hash` or
inline in the service package. bcrypt cost = `bcrypt.DefaultCost`.

## 7. Infrastructure

- `internal/infrastructure/token/jwt.go` — `JWT` implements `TokenIssuer`
  using `golang-jwt/jwt/v5`, HS256. Access claims `typ:"admin"`; pending
  claims `typ:"admin_2fa_pending"`. `ParseAccess` rejects wrong `typ` / expiry.
  `IssueAccess` returns `expiresIn` seconds from cfg.
- `internal/infrastructure/sms/fake.go`:
  ```go
  // FakeSender logs the message instead of sending it. Every code it "delivers"
  // is the fixed value sms.FixedCode.
  // TODO: implement the real sms.ir provider (https://sms.ir) — HTTP client,
  // API key + line number from config, template send. Swap this out in main.go.
  const FixedCode = "123456"
  type FakeSender struct{ Log *log.Logger }
  func (f FakeSender) Send(ctx context.Context, phone, message string) error { ... log ... return nil }
  ```
  `CodeGenerator` for wiring: fake = `func() string { return sms.FixedCode }`.
  Add a `// TODO: real generator = crypto/rand 6-digit` note.
- `internal/infrastructure/repo/admin_repo.go` — `AdminRepo` wraps
  `*sqlc.Queries` (+ pool), implements `adminauth.Repository`. Map
  `pgx.ErrNoRows` to the right domain/sentinel errors. Convert sqlc row types
  ↔ `admin.Admin` and the small `RefreshToken` / `PasswordReset` DTOs.

## 8. Interface (HTTP) — `api/internal/interface/http/`

- `middleware.go`:
  - `CORS(origins []string)` — echo `Origin` if in allowlist, set
    `Access-Control-Allow-Credentials: true`, `-Allow-Methods`,
    `-Allow-Headers: Authorization,Content-Type`, `-Max-Age`. Short-circuit
    `OPTIONS` with 204.
  - `AdminAuth(tokens TokenIssuer)` — read `Authorization: Bearer`, `ParseAccess`,
    set `adminID` in context; 401 otherwise.
- `admin_auth_handler.go` — handlers + request/response structs with binding
  tags. Envelope: success returns the payload directly; errors return
  `{"error": {"code": "...", "message": "..."}}` with proper status
  (pick codes: `invalid_credentials` 401, `twofa_required` is a normal 200
  body not an error, `twofa_invalid` 401, `reset_code_invalid` 400,
  `refresh_invalid` 401, `validation_error` 400). Keep an
  `errorResponse(c, status, code, msg)` helper. This is the project's first
  error envelope — keep it minimal, note it in `api/CLAUDE.md`.
  - Cookie helper `setRefreshCookie(c, raw, cfg)` /
    `clearRefreshCookie(c, cfg)`: name `admin_refresh_token`, `HttpOnly`,
    `Secure=cfg.CookieSecure`, `SameSite=None` (cross-origin SPA;
    `http.SameSiteNoneMode`), `Path=/admin/auth`, `Domain=cfg.CookieDomain`,
    `MaxAge` = refresh TTL seconds. Read back with
    `c.Request.Cookie("admin_refresh_token")`.
    ponytail note comment: `SameSite=None requires Secure; browsers treat
    http://localhost as a secure context so dev over http still works`.
- `router.go` — `NewRouter` gains params (health svc + adminauth svc + tokens
  + cfg). Register:
  ```
  r.Use(CORS(cfg.CORSOrigins))
  r.GET  /health
  g := r.Group("/admin/auth")
    POST /login
    POST /login/2fa
    POST /refresh
    POST /logout
    POST /forgot-password
    POST /reset-password
  p := r.Group("/admin"); p.Use(AdminAuth(tokens))
    GET  /me
    POST /2fa/setup
    POST /2fa/activate
    POST /2fa/disable
  ```
- `cmd/server/main.go` — build the sqlc `Queries`, repo, JWT issuer, fake SMS
  sender + code generator, `adminauth.NewService(...)`, pass into `NewRouter`.

### Endpoint contracts

| Method + path | Body | Success | Errors |
|---|---|---|---|
| `POST /admin/auth/login` | `{phone_number,password}` | `200 {twofa_required:false, access_token, expires_in, admin}` + Set-Cookie; or `200 {twofa_required:true, pending_token}` | 401 invalid_credentials, 400 validation_error |
| `POST /admin/auth/login/2fa` | `{pending_token,code}` | `200 {access_token, expires_in, admin}` + Set-Cookie | 401 twofa_invalid |
| `POST /admin/auth/refresh` | — (cookie) | `200 {access_token, expires_in}` + rotated Set-Cookie | 401 refresh_invalid |
| `POST /admin/auth/logout` | — (cookie) | `204` + cleared cookie | always 204 |
| `POST /admin/auth/forgot-password` | `{phone_number}` | `200 {message}` (generic) | 400 validation_error |
| `POST /admin/auth/reset-password` | `{phone_number,code,new_password}` | `200 {message}` | 400 reset_code_invalid / validation_error |
| `GET /admin/me` | Bearer | `200 {admin}` | 401 |
| `POST /admin/2fa/setup` | Bearer | `200 {secret, otpauth_url}` | 401 |
| `POST /admin/2fa/activate` | Bearer `{secret,code}` | `200 {enabled:true}` | 401, 400 twofa_invalid |
| `POST /admin/2fa/disable` | Bearer `{password}` | `200 {enabled:false}` | 401 invalid_credentials |

`admin` JSON = `{id, first_name, last_name, email, phone_number, is_male, two_fa_enabled, created_at}`. Never expose `password_hash` / `totp_secret`.

## 9. Seeder — `api/cmd/seed/main.go`

Load config, open pgx pool, `bcrypt` the seed password, run an upsert
(`INSERT ... ON CONFLICT (phone_number) DO UPDATE SET first_name=EXCLUDED.first_name,
last_name=EXCLUDED.last_name, email=EXCLUDED.email, password_hash=EXCLUDED.password_hash`).
Log `seeded admin id=<n> phone=9370843199`. Exit 0 on success, non-zero on
error. Must be safe to run repeatedly.

## 10. Makefile — `api/Makefile`

```make
DATABASE_URL ?= postgres://hesab:hesab@localhost:5432/hesab?sslmode=disable
migrate-up:      ; migrate -path db/migration -database "$(DATABASE_URL)" up
migrate-down:    ; migrate -path db/migration -database "$(DATABASE_URL)" down 1
migrate-create:  ; migrate create -ext sql -dir db/migration -seq $(name)
sqlc:            ; sqlc generate
seed:            ; go run ./cmd/seed
run:             ; go run ./cmd/server
test:            ; go test ./...
```

## 11. Tests (ponytail: one runnable check per non-trivial unit — asserts, no framework)

- `internal/domain/admin/phone_test.go` — table test: `+989370843199`,
  `00989370843199`, `09370843199`, `9370843199`, `"937 084 3199"`,
  `"0937-084-3199"` all → `9370843199`; `"12345"`, `"8370843199"`, `""` → error.
- `internal/infrastructure/token/jwt_test.go` — access round-trip; expired
  token rejected; pending token rejected by `ParseAccess` and vice versa.
- `internal/application/adminauth/service_test.go` — in-memory fake `Repository`,
  real `JWT` issuer, real `pquerna/otp`, `CodeGenerator` = `() => "123456"`,
  fixed `Clock`:
  - login no-2FA → access + refresh returned;
  - login wrong password → `ErrInvalidCredentials`;
  - login with 2FA on → `TwoFARequired`, no tokens; then `LoginVerify2FA` with
    `totp.GenerateCode(secret, now)` → tokens;
  - `Refresh` rotates: old raw refresh now fails, new one works;
  - `Logout` then `Refresh` → `ErrRefreshInvalid`;
  - `ForgotPassword` then `ResetPassword` with `"123456"` → password changed,
    all refresh tokens revoked; wrong code → `ErrResetCodeInvalid`.

## 12. Frontend — `admin/` (Next.js 16, App Router, `output: "export"`, RTL, dark)

Run the **ui-ux-pro-max** skill for design direction first
(`export PYTHONIOENCODING=utf-8` before its script — documented Windows fix).
Reuse `--color-brand-*` tokens in `admin/app/globals.css`, Vazirmatn font,
`dir="rtl"` (already in `layout.tsx`). All pages are client components
(`"use client"`) — static export has no server runtime.

### Shared libs

- `admin/lib/api.ts` — `API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"`.
  `apiFetch(path, {method, body, auth})`: JSON in/out, `credentials: "include"`,
  attaches `Authorization: Bearer <accessToken>` when `auth`. On `401` with
  `auth`, call `/admin/auth/refresh` once, store new token, retry once.
- `admin/lib/auth.ts` — module-level `accessToken` var, mirrored to
  `sessionStorage` (`admin_access_token`) so a reload survives until refresh.
  `pendingToken` in `sessionStorage` (`admin_pending_token`). Helpers:
  `setAccessToken`, `getAccessToken`, `clearSession`, `setPending`, `takePending`.
- `admin/lib/useRequireAuth.ts` — client hook: on mount, if no access token
  try `apiFetch("/admin/me",{auth:true})` (which triggers a refresh); on
  failure `router.replace("/login")`. Returns `{admin, loading}`.

### Pages

| Route | Content |
|---|---|
| `app/login/page.tsx` | Phone + password fields (Persian labels: «شماره موبایل», «رمز عبور»). Submit → `POST /admin/auth/login`. `twofa_required` → `setPending`, `router.push("/login/2fa")`. else `setAccessToken`, `router.replace("/")`. Link to `/forgot-password`. Client-side phone normalize + inline errors. |
| `app/login/2fa/page.tsx` | Single 6-digit code input («کد تأیید دو مرحله‌ای»). `takePending()`; if none → back to `/login`. Submit → `POST /admin/auth/login/2fa` → `setAccessToken`, `router.replace("/")`. |
| `app/forgot-password/page.tsx` | Phone field. Submit → `POST /admin/auth/forgot-password` → always show «اگر شماره ثبت شده باشد، کد ارسال شد» then `router.push("/reset-password?phone=<normalized>")`. |
| `app/reset-password/page.tsx` | Fields: phone (prefilled from `?phone=`), code, new password, confirm. Submit → `POST /admin/auth/reset-password` → on success toast + `router.replace("/login")`. |
| `app/page.tsx` | Replace the current dummy login. Now the protected dashboard shell: `useRequireAuth()`; while loading show a spinner; render greeting «خوش آمدید، {first_name}», a logout button (`POST /admin/auth/logout` → `clearSession` → `/login`), and a link to `/settings/security`. |
| `app/settings/security/page.tsx` | `useRequireAuth()`. If `admin.two_fa_enabled` false: button «فعال‌سازی ورود دو مرحله‌ای» → `POST /admin/2fa/setup` → show `secret` (monospace, copyable) + the `otpauth_url` as a link, plus a code input → `POST /admin/2fa/activate {secret, code}` → success, refetch `/admin/me`. If enabled: password field + «غیرفعال‌سازی» → `POST /admin/2fa/disable {password}`. ponytail: show the secret string + otpauth link as text, **no QR image / no qrcode dependency** — note in a comment that a QR can be added later. |

Keep styling consistent with the existing login page's look (dark `#0F172A`
bg, surface `#1E293B`, amber `#F59E0B` accent, rounded cards, Vazirmatn).
Centered card layout for the auth pages; simple settings layout for the panel.

### Frontend acceptance

`cd admin && npm run build` → static export, **0 errors**. All six routes
prerender. No new runtime dependency added (dev-only types are fine).

---

## Global acceptance criteria

1. `cd api && go build ./... && go test ./...` — clean.
2. `cd api && make migrate-up` — creates `admins`,
   `admin_refresh_tokens`, `admin_password_resets`. `make migrate-down` reverses.
3. `cd api && make seed` — inserts the seed admin; second run succeeds (idempotent).
4. `POST /admin/auth/login {"phone_number":"9370843199","password":"Amir@Pass1999"}`
   → `200`, `twofa_required:false`, `access_token` set, `Set-Cookie: admin_refresh_token`.
5. Wrong password → `401 invalid_credentials`.
6. `POST /admin/auth/refresh` with the cookie → `200` new access token + new
   cookie; replaying the **old** refresh cookie → `401`.
7. `POST /admin/auth/logout` → `204`; subsequent `refresh` → `401`.
8. `POST /admin/auth/forgot-password {"phone_number":"9370843199"}` → `200`;
   server log shows the fake SMS carrying code `123456`.
9. `POST /admin/auth/reset-password {"phone_number":"9370843199","code":"123456",
   "new_password":"NewPass2026"}` → `200`; login with the new password works;
   pre-existing refresh tokens all revoked.
10. 2FA: `setup` → secret + otpauth_url; `activate` with a valid TOTP code →
    `enabled:true`; next `login` → `twofa_required:true` + `pending_token`;
    `login/2fa` with a valid code → `200` tokens; `disable` with password →
    `enabled:false`.
11. `cd admin && npm run build` → 0 errors; `/login`, `/login/2fa`,
    `/forgot-password`, `/reset-password`, `/settings/security` all build.
12. CORS preflight (`OPTIONS`) from `http://localhost:3010` returns
    `Access-Control-Allow-Origin: http://localhost:3010` +
    `Access-Control-Allow-Credentials: true`.

## After implementation — update learnings

- `api/CLAUDE.md` «Learnings»: error envelope shape; phone normalization rule;
  refresh-token rotation + cookie (`SameSite=None`, localhost-secure caveat);
  golang-migrate is CLI-only, migrations in `api/db/migration`; seeder =
  `make seed`; SMS is a fake stub — real sms.ir is a TODO.
- `admin/CLAUDE.md` «Learnings»: access token in memory + `sessionStorage`,
  refresh via HttpOnly cookie; `apiFetch` auto-refresh-once; auth pages are
  client components; `NEXT_PUBLIC_API_URL` env.
