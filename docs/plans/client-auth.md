# Plan — Client dashboard authentication (`./client` + `./api`)

## Context

Admin auth already exists and is committed (`/admin/auth/*`, `/admin/2fa/*`,
6 RTL admin pages, seeder, fake SMS). **Do not touch admin code.** This task
builds the *same* auth for the **client dashboard** (`./client`, companies /
individuals), mirroring the admin implementation as closely as possible.

Decisions already made (do not re-ask):

- Table is `users` (not `clients`). Columns exactly: `id`, `first_name`,
  `last_name`, `email`, `phone_number`, `password_hash`, `totp_secret`,
  `created_at`. **No `is_male`** (admin has it; users do not).
- Auth is by **phone number**, never email. Same normalization as admin
  (`9XXXXXXXXX` canonical).
- Client users are **provisioned by admin**, no public signup page. For this
  task there is no admin UI to create them yet — add a `CreateUser` sqlc query
  for later, and seed one test user so client login is testable.
- Client 2FA works **exactly like admin**: user self-enables TOTP in
  settings; when enabled, login is two-step (password → `pending_token` →
  TOTP code → tokens).
- SMS stays the existing fake fixed-code `123456` stub (`internal/infrastructure/sms`).
  Do not implement sms.ir — the TODO already documents it. Reuse `sms.FakeSender`
  and `sms.FixedCode` as-is.
- Refresh token: opaque 32-byte random, stored SHA-256 hashed, rotated every
  refresh, revoked on logout and on password reset. Same as admin.
- Reuse existing config TTLs (`ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`,
  `PASSWORD_RESET_TTL`, `TWOFA_PENDING_TTL`).

The overall approach is **copy the proven admin module to a parallel client
module**, changing only names and table targets. Duplication is deliberate and
sanctioned here — do not build a shared generic auth core. Mark the top of each
new file that is a near-copy with `// ponytail: parallel to adminauth, kept
separate on purpose`.

---

## Backend (`./api`)

### 1. Migration — `db/migration/000002_client_auth.up.sql` / `.down.sql`

Mirror `000001_admin_auth.up.sql` with `users` / `user_refresh_tokens` /
`user_password_resets`:

```sql
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    phone_number TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    totp_secret TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_refresh_tokens (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_refresh_tokens_user_id_idx ON user_refresh_tokens (user_id);

CREATE TABLE user_password_resets (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_password_resets_user_id_idx ON user_password_resets (user_id);
```

`.down.sql` drops the three tables in reverse order.

### 2. sqlc queries — `query/client_auth.sql`

Mirror `query/admin_auth.sql`, retargeted:

- `GetUserByPhone :one`, `GetUserByID :one`
- `CreateUser :one` — `INSERT INTO users (first_name,last_name,email,phone_number,password_hash,totp_secret) VALUES ($1,$2,$3,$4,$5,$6) RETURNING *;`
- `UpdateUserPassword :exec`, `SetUserTOTPSecret :exec`
- `InsertUserRefreshToken :one`, `GetUserRefreshToken :one`,
  `RevokeUserRefreshToken :exec`, `RevokeAllUserRefreshTokens :exec`
- `InsertUserPasswordReset :one`, `InvalidateUserPasswordResets :exec`,
  `GetLatestUserPasswordReset :one`, `ConsumeUserPasswordReset :exec`

Then run `sqlc generate` from `./api` (binary at `~/go/bin/sqlc`). Commit the
regenerated `internal/infrastructure/db/sqlc/*` files.

### 3. Domain — `internal/domain/user/`

- `user.go` — `User` struct (`ID int64`; `FirstName,LastName,Email,PhoneNumber,PasswordHash,TOTPSecret string`;
  `CreatedAt time.Time`), `func (u User) TwoFAEnabled() bool`, and the same
  error vars as `domain/admin` (`ErrInvalidCredentials`, `ErrResetCodeInvalid`,
  `ErrRefreshInvalid`, `ErrTwoFARequired`, `ErrTwoFACodeInvalid`,
  `ErrWeakPassword`).
- `phone.go` — copy `domain/admin/phone.go` verbatim, package `user`.
- `password.go` — copy `domain/admin/password.go` verbatim, package `user`.
- `phone_test.go` — copy admin's.

### 4. Token — `internal/infrastructure/token/jwt.go`

Add client methods alongside the admin ones (same `JWT` struct, same secret,
same TTLs):

- `IssueClientAccess(id) (string,int,error)` — typ `"client"`
- `IssueClientPending(id) (string,error)` — typ `"client_2fa_pending"`
- `ParseClientAccess(s) (int64,error)` / `ParseClientPending(s) (int64,error)`

### 5. Application — `internal/application/clientauth/service.go`

Near-verbatim copy of `internal/application/adminauth/service.go`:

- `s/adminauth/clientauth/`, `s/admin\./user\./` (domain pkg), `s/Admin/User/`,
  `s/AdminID/UserID/` in the local `RefreshToken` / `PasswordReset` structs and
  `Repository` interface.
- `TokenIssuer` interface here declares `IssueAccess/IssuePending/ParseAccess/ParsePending`
  — wire the concrete `token.JWT` to it in `main.go` via a thin adapter
  (see step 8) OR add a `clientTokenAdapter` struct in `main.go` that maps
  `IssueAccess -> IssueClientAccess` etc. Keep the adapter in `main.go`, do
  not export it.
- `SMSSender`, `CodeGenerator`, `Clock` identical.
- Same methods: `Login`, `LoginVerify2FA`, `Refresh`, `Logout`,
  `ForgotPassword`, `ResetPassword`, `Setup2FA`, `Activate2FA`, `Disable2FA`,
  `Me`. Password-reset SMS text identical (`c+" کد بازیابی رمز عبور شماست"`).
- `Setup2FA` issuer: use `cfg.ClientTOTPIssuer` (new, see step 7).
- Copy `service_test.go` too, retargeted; it must pass.

### 6. Infrastructure repo — `internal/infrastructure/repo/user_repo.go`

Mirror `admin_repo.go` → `UserRepo` implementing `clientauth.Repository`,
using the new sqlc methods. `userFrom(sqlc.User) user.User` (no `IsMale`).

### 7. Config — `internal/config/config.go`

Add one field: `ClientTOTPIssuer string`, default
`getenv("CLIENT_TOTP_ISSUER", "Hesab")` (admin's is `"Hesab Admin"`).
Nothing else changes — cookie secure/domain and CORS are shared.

### 8. HTTP wiring

- `internal/interface/http/middleware.go` — add `ClientAuth(t clientauth.TokenIssuer) gin.HandlerFunc`,
  identical to `AdminAuth` but sets `c.Set("userID", id)`.
- `internal/interface/http/client_auth_handler.go` — mirror
  `admin_auth_handler.go`:
  - `clientAuthHandler{svc *clientauth.Service; tokens clientauth.TokenIssuer; cfg config.Config}`
  - request structs can be reused from the package (`loginReq`, `twoReq`,
    `forgotReq`, `resetReq`, `activateReq`, `disableReq` already exist in
    package `http` — reuse them, do not redeclare).
  - `userJSON(u user.User) gin.H` — same keys as `adminJSON` minus `is_male`:
    `id, first_name, last_name, email, phone_number, two_fa_enabled, created_at`.
  - Cookie: name `client_refresh_token`, path `/client/auth`, same
    SameSite=None / Secure / HttpOnly logic. Separate `setCookie` /
    `clearCookie` / `refresh` helpers (name them `setClientCookie` etc. to
    avoid clashing with the admin helpers in the same package).
  - Persian error/success strings identical to admin handler.
- `internal/interface/http/router.go` — `NewRouter` signature gains
  `clientSvc *clientauth.Service, clientTokens clientauth.TokenIssuer`.
  Register:
  ```
  cg := r.Group("/client/auth")
  {
      cg.POST("/login", ca.login)
      cg.POST("/login/2fa", ca.login2fa)
      cg.POST("/refresh", ca.refresh)
      cg.POST("/logout", ca.logout)
      cg.POST("/forgot-password", ca.forgot)
      cg.POST("/reset-password", ca.reset)
  }
  cp := r.Group("/client")
  cp.Use(ClientAuth(clientTokens))
  {
      cp.GET("/me", ca.me)
      cp.POST("/2fa/setup", ca.setup)
      cp.POST("/2fa/activate", ca.activate)
      cp.POST("/2fa/disable", ca.disable)
  }
  ```
- `cmd/server/main.go` — build `clientauth.NewService(repo.NewUserRepo(sqlc.New(pool)),
  clientTokenAdapter{tokens}, sms.FakeSender{Log: log.Default()},
  func() string { return sms.FixedCode }, time.Now, cfg)` and pass it +
  the adapter into `NewRouter`. Define `clientTokenAdapter` here.

### 9. Seeder — `cmd/seed/main.go`

Keep the existing admin insert (`9370843199` / `Amir@Pass1999`) untouched.
Append a client test user so client login is testable:

- phone `9120000000`, password `Client@Pass1999`,
  first/last `تست` / `کاربر`, email `user@hesab.local`, `totp_secret=''`.
- Same `ON CONFLICT (phone_number) DO UPDATE` upsert shape as the admin insert,
  targeting `users`.
- Log `seeded user id=%d phone=9120000000`.

---

## Frontend (`./client`)

Mirror the admin SPA pages. Client is **light mode, blue accent** — the brand
tokens in `client/app/globals.css` are already set; just use `bg-brand-*`,
`text-brand-*`, `border-brand-*`, `bg-brand-accent`, etc. Every page keeps
`lang="fa" dir="rtl"` (root layout already sets it). `sonner` is already a
dependency; add `<Toaster />` to the root layout if not present (copy admin's
layout treatment).

Copy these admin files to `./client` and retarget every endpoint from
`/admin/...` to `/client/...`, storage keys from `admin_*` to `client_*`,
the `Admin` type to `User` (drop `is_male`), and the refresh path to
`/client/auth/refresh`:

| Admin source | Client target |
|---|---|
| `admin/lib/api.ts` | `client/lib/api.ts` (refresh URL `/client/auth/refresh`) |
| `admin/lib/auth.ts` | `client/lib/auth.ts` (keys `client_access_token`, `client_pending_token`) |
| `admin/lib/useRequireAuth.ts` | `client/lib/useRequireAuth.ts` (`User` type, `GET /client/me`) |
| `admin/app/login/page.tsx` | `client/app/login/page.tsx` |
| `admin/app/login/2fa/page.tsx` | `client/app/login/2fa/page.tsx` |
| `admin/app/forgot-password/page.tsx` | `client/app/forgot-password/page.tsx` |
| `admin/app/reset-password/page.tsx` | `client/app/reset-password/page.tsx` |
| `admin/app/settings/security/page.tsx` | `client/app/settings/security/page.tsx` |
| `admin/app/page.tsx` (dashboard) | `client/app/page.tsx` (replace the current dummy page) |

Copy prose/labels as-is except swap the admin-panel wording for
client-dashboard wording (e.g. "پنل مدیریت" → "داشبورد", the left-hero
marketing copy → a short companies/individuals accounting line). Keep it
minimal — do not redesign, do not add screens beyond the table above.

`client/app/page.tsx` must: `useRequireAuth()`, show the signed-in user's
name, a logout button (`POST /client/auth/logout` then `clearSession()` then
`router.replace("/login")`), and a link to `/settings/security`.

Check `client/next.config.ts` already pins `turbopack: { root: import.meta.dirname }`
and `output: "export"` — leave as-is. If `NEXT_PUBLIC_API_URL` is used by
`admin/lib/api.ts`, mirror the same env handling in `client/lib/api.ts`
(default `http://localhost:8080`).

---

## Env / compose

- `.env.example` — add `CLIENT_TOTP_ISSUER=Hesab` near `TOTP_ISSUER`.
- `docker-compose.yml` — pass `CLIENT_TOTP_ISSUER` through to the `api`
  service env if `TOTP_ISSUER` is already listed there; otherwise no change.
- CORS already allows `http://localhost:3020` — no change.

---

## Acceptance criteria

1. `cd api && go build ./... && go test ./...` — all pass (incl. new
   `clientauth` + `domain/user` tests).
2. `cd api && sqlc generate` — no diff after commit (generated code committed).
3. `migrate -path db/migration -database "$DATABASE_URL" up` applies `000002`
   cleanly; `.down.sql` reverses it.
4. `go run ./cmd/seed` — seeds admin (unchanged) **and** client user
   `9120000000` / `Client@Pass1999`.
5. Manual API check (server running, DB migrated + seeded):
   - `POST /client/auth/login` `{phone_number:"9120000000",password:"Client@Pass1999"}`
     → `200`, `access_token`, `Set-Cookie: client_refresh_token=...`.
   - `POST /client/auth/forgot-password` → `200`; `POST /client/auth/reset-password`
     with `code:"123456"` → `200`; old refresh tokens revoked.
   - `POST /client/auth/refresh` (with cookie) → new access token + rotated cookie.
   - `POST /client/2fa/setup` → secret; `POST /client/2fa/activate` with a
     valid TOTP for that secret → `{enabled:true}`; subsequent login returns
     `twofa_required:true` + `pending_token`, then `POST /client/auth/login/2fa`
     → tokens.
   - `POST /client/auth/logout` → `204`, cookie cleared.
   - Admin endpoints (`/admin/auth/*`) still behave exactly as before.
6. `cd client && npm run build` — static export, 0 errors. New pages serve
   `lang="fa" dir="rtl"`, light theme.

## After coding

Append a dated bullet to root `CLAUDE.md` "Learnings log" and to
`api/CLAUDE.md` "Learnings" summarizing the client-auth stack (parallel to
adminauth, `users` table, `/client/auth/*` routes, `client_refresh_token`
cookie on path `/client/auth`, seeded test user).
