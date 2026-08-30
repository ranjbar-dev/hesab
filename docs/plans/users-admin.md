# Plan — Admin user management (`./admin` + `./api`)

## Context

`feat/client-auth` is **already merged to main**. The `users` table, the
`user` domain package, `clientauth` application service, `repo.UserRepo`,
`query/client_auth.sql`, and `/client/auth/*` routes all exist. This task adds
the **super-admin CRUD** on top of that same `users` table:

- list users in a table with a per-column filter row that re-fetches from the
  API on every change (debounced),
- create users from a modal on the list page,
- a `/users/[id]` detail page with edit / enable-disable / reset-password /
  soft-delete,
- one phone number → at most one active user account.

**Do not touch** `clientauth`, `admin_auth`, or the `/client` frontend.
Reuse existing patterns and styles everywhere.

### Decisions already made (do not re-ask)

- User entity = the existing `users` table (client dashboard end-users).
- Name fields: keep the existing `first_name` + `last_name` (NOT `full_name`).
- Fields the admin manages: `first_name`, `last_name`, `phone_number` (login
  id, immutable after create), `email` (optional), `national_id` (optional,
  Iranian 10-digit کد ملی, validated by checksum, **not unique**),
  `account_type` (`individual` | `company`, flat label only), `password` (set
  at create + resettable), `status` (`active` | `disabled`), `created_at`.
- Creating a user sets real credentials (phone + password) and "sends" a
  welcome SMS — SMS stays the existing fake no-op stub (`sms.FakeSender`).
- Soft everything: `disable` = `status='disabled'` (blocks future client
  login once enforced); `delete` = set `deleted_at`, row disappears from all
  admin lists, nothing is hard-deleted.
- Phone uniqueness: the table already has a hard `UNIQUE(phone_number)`. Keep
  it. A soft-deleted user keeps its phone reserved — recreating with a
  previously used phone returns a 409. Acceptable for now.
- Filter columns: `first_name`, `last_name`, `phone` (debounced substring),
  `status` (select), `created_at` (single Jalali day, exact-day match).
  No filter on `account_type` (shown as a column only).
- List is server-paginated: `page` (1-based), `page_size` (default 20).
- Detail-page actions: edit profile, enable/disable toggle, reset password,
  soft-delete. **No** "resend SMS" button.

---

## Backend (`./api`)

### 1. Migration — `db/migration/000003_users_admin.up.sql` / `.down.sql`

New columns are additive so existing `sqlc.User` consumers (`repo.UserRepo`,
`clientauth`) keep compiling. Only `email` changes shape, and only by dropping
its `UNIQUE` + giving it a default — its Go type stays `string`.

`000003_users_admin.up.sql`:

```sql
ALTER TABLE users
    ADD COLUMN national_id  TEXT,
    ADD COLUMN account_type TEXT NOT NULL DEFAULT 'individual',
    ADD COLUMN status       TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN deleted_at   TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT users_email_key;
ALTER TABLE users ALTER COLUMN email SET DEFAULT '';

CREATE INDEX users_created_at_idx ON users (created_at DESC);
CREATE INDEX users_status_idx     ON users (status) WHERE deleted_at IS NULL;
```

`000003_users_admin.down.sql`:

```sql
DROP INDEX users_status_idx;
DROP INDEX users_created_at_idx;
ALTER TABLE users ALTER COLUMN email DROP DEFAULT;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE users
    DROP COLUMN deleted_at,
    DROP COLUMN status,
    DROP COLUMN account_type,
    DROP COLUMN national_id;
```

`email` stays `TEXT NOT NULL` — when the admin leaves it blank we store `''`,
never `NULL`. That keeps the generated `Email string` field unchanged.

### 2. Queries — `query/users_admin.sql` (new file)

All `admin`-prefixed so they never clash with client-auth query names. Use
`sqlc.narg` for every optional filter and `sqlc.arg` for limit/offset.

```sql
-- name: AdminCreateUser :one
INSERT INTO users (first_name, last_name, email, phone_number, national_id, account_type, password_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: AdminGetUser :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: AdminListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('first_name')::text  IS NULL OR first_name   ILIKE '%' || sqlc.narg('first_name') || '%')
  AND (sqlc.narg('last_name')::text   IS NULL OR last_name    ILIKE '%' || sqlc.narg('last_name')  || '%')
  AND (sqlc.narg('phone')::text       IS NULL OR phone_number ILIKE '%' || sqlc.narg('phone')      || '%')
  AND (sqlc.narg('status')::text      IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz   IS NULL OR created_at <  sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: AdminCountUsers :one
SELECT count(*) FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('first_name')::text  IS NULL OR first_name   ILIKE '%' || sqlc.narg('first_name') || '%')
  AND (sqlc.narg('last_name')::text   IS NULL OR last_name    ILIKE '%' || sqlc.narg('last_name')  || '%')
  AND (sqlc.narg('phone')::text       IS NULL OR phone_number ILIKE '%' || sqlc.narg('phone')      || '%')
  AND (sqlc.narg('status')::text      IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz   IS NULL OR created_at <  sqlc.narg('created_to'));

-- name: AdminUpdateUserProfile :one
UPDATE users SET first_name = $2, last_name = $3, email = $4, national_id = $5, account_type = $6
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: AdminSetUserStatus :one
UPDATE users SET status = $2 WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: AdminSetUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: AdminSoftDeleteUser :exec
UPDATE users SET deleted_at = now(), status = 'disabled' WHERE id = $1 AND deleted_at IS NULL;
```

Session revocation on disable / reset / delete reuses the **existing**
`RevokeAllUserRefreshTokens` query — do not add a new one.

Run `sqlc generate` in `./api` (`~/go/bin/sqlc` if present, else
`go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate`). Commit the
regenerated files.

### 3. Domain — `internal/domain/user/user_admin.go` (new file)

```go
package user

import (
	"errors"
	"strconv"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	AccountIndividual = "individual"
	AccountCompany    = "company"
)

var (
	ErrPhoneTaken         = errors.New("phone already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidAccountType = errors.New("invalid account type")
	ErrInvalidNationalID  = errors.New("invalid national id")
)

func ValidStatus(s string) bool      { return s == StatusActive || s == StatusDisabled }
func ValidAccountType(s string) bool { return s == AccountIndividual || s == AccountCompany }

// ValidateNationalID accepts an empty string (the field is optional). A
// non-empty value must be a well-formed Iranian national code (کد ملی):
// 10 digits, not all-identical, passing the standard check-digit test.
func ValidateNationalID(s string) error {
	if s == "" {
		return nil
	}
	if len(s) != 10 {
		return ErrInvalidNationalID
	}
	allSame := true
	sum := 0
	for i := 0; i < 9; i++ {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			return ErrInvalidNationalID
		}
		if s[i] != s[0] {
			allSame = false
		}
		sum += d * (10 - i)
	}
	check, err := strconv.Atoi(s[9:])
	if err != nil || allSame {
		return ErrInvalidNationalID
	}
	r := sum % 11
	if (r < 2 && check == r) || (r >= 2 && check == 11-r) {
		return nil
	}
	return ErrInvalidNationalID
}
```

Extend the shared `User` struct in `internal/domain/user/user.go` with the
new fields (additive — client-auth ignores them):

```go
type User struct {
	ID                                                                int64
	FirstName, LastName, Email, PhoneNumber, PasswordHash, TOTPSecret string
	NationalID, AccountType, Status                                   string
	DeletedAt                                                         *time.Time
	CreatedAt                                                         time.Time
}
```

Leave `repo.UserRepo.userFrom` as-is (it just won't populate the new fields;
client-auth doesn't read them). The admin repo gets its own mapper.

### 4. Repo — `internal/infrastructure/repo/user_admin_repo.go` (new file)

`type UserAdminRepo struct{ q *sqlc.Queries }` + `NewUserAdminRepo`. Wraps the
new queries. Responsibilities:

- map `sqlc.User` → `user.User` (a local `userAdminRow` mapper handling the
  nullable `national_id` / `deleted_at` pgtype values),
- `pgx.ErrNoRows` → `user.ErrUserNotFound`,
- Postgres unique violation on create → `user.ErrPhoneTaken`
  (`var pgErr *pgconn.PgError; errors.As(err, &pgErr) && pgErr.Code == "23505"`),
- a `ListFilter` struct (`FirstName, LastName, Phone, Status *string;
  CreatedFrom, CreatedTo *time.Time; Limit, Offset int32`) → maps to the
  sqlc params (`nil` pointer → SQL `NULL`),
- `List` returns `([]user.User, error)`; `Count` returns `(int64, error)`.
- `RevokeAllSessions(ctx, id)` → `q.RevokeAllUserRefreshTokens`.

Method set: `Create`, `Get`, `List`, `Count`, `UpdateProfile`, `SetStatus`,
`SetPassword`, `SoftDelete`, `RevokeAllSessions`.

### 5. Application — `internal/application/usersadmin/service.go` (new package)

```go
package usersadmin
```

Interfaces (small, local):

```go
type Repository interface {
	Create(ctx context.Context, in NewUser) (user.User, error)
	Get(ctx context.Context, id int64) (user.User, error)
	List(ctx context.Context, f ListFilter) ([]user.User, error)
	Count(ctx context.Context, f ListFilter) (int64, error)
	UpdateProfile(ctx context.Context, id int64, in Profile) (user.User, error)
	SetStatus(ctx context.Context, id int64, status string) (user.User, error)
	SetPassword(ctx context.Context, id int64, hash string) error
	SoftDelete(ctx context.Context, id int64) error
	RevokeAllSessions(ctx context.Context, id int64) error
}
type SMS interface { Send(ctx context.Context, phone, message string) error }
```

`Service` holds `repo Repository`, `sms SMS`. `NewService(repo, sms)`.

Use cases:

- **`Create(ctx, input)`** — `user.NormalizePhone`; require `first_name`,
  `last_name`; `user.ValidatePassword`; `user.ValidateNationalID`; default
  `account_type` to `individual`, else `user.ValidAccountType`;
  `bcrypt.GenerateFromPassword(..., bcrypt.DefaultCost)`; `repo.Create`.
  On `user.ErrPhoneTaken` return it unchanged. On success, best-effort
  `sms.Send(ctx, phone, welcomeMessage)` — log and swallow any error, never
  fail the create because SMS failed.
  `welcomeMessage` = `"حساب کاربری شما در سامانه حساب ساخته شد. برای ورود از شماره موبایل خود استفاده کنید."`
  `// TODO(sms): real template + provider once sms.ir lands`.
- **`List(ctx, filter, page, pageSize)`** — clamp `pageSize` to `1..100`
  (default 20 when ≤0), `page` to `≥1`; `offset = (page-1)*pageSize`; call
  `repo.List` + `repo.Count`; return `(users, total)`.
- **`Get(ctx, id)`** — passthrough; `ErrUserNotFound` bubbles.
- **`UpdateProfile(ctx, id, input)`** — same field validation as create
  (names required, national id, account type); `repo.UpdateProfile`.
- **`SetStatus(ctx, id, status)`** — `user.ValidStatus`; `repo.SetStatus`;
  if `status == disabled` also `repo.RevokeAllSessions(ctx, id)` (kill live
  client sessions — security).
- **`ResetPassword(ctx, id, newPassword)`** — `user.ValidatePassword`;
  bcrypt; `repo.SetPassword`; then `repo.RevokeAllSessions(ctx, id)`
  (force re-login — security).
- **`Delete(ctx, id)`** — `repo.SoftDelete`; then `repo.RevokeAllSessions`.

Input structs (`NewUser`, `Profile`, `ListFilter`) live in this package;
`ListFilter` mirrors the repo one but with plain values + a bool/`*T` per
optional filter — keep it simple, the handler builds it from query params.

### 6. HTTP — `internal/interface/http/users_admin_handler.go` (new file)

`type usersAdminHandler struct{ svc *usersadmin.Service }`. Reuse the existing
`errJSON(c, status, code, msg)` helper. Add:

```go
func userJSON(u user.User) gin.H {
	return gin.H{
		"id": u.ID, "first_name": u.FirstName, "last_name": u.LastName,
		"email": u.Email, "phone_number": u.PhoneNumber,
		"national_id": u.NationalID, "account_type": u.AccountType,
		"status": u.Status, "created_at": u.CreatedAt,
	}
}
```

Endpoints (all under the existing authenticated `/admin` group — see router
change below):

| Method + path | Body | Success | Errors |
|---|---|---|---|
| `GET /admin/users` | — (query params) | `200 {users:[...], total, page, page_size}` | `400 validation_error` |
| `POST /admin/users` | `{first_name,last_name,email?,phone_number,national_id?,account_type?,password}` | `201 {user}` | `400 validation_error`, `409 phone_taken` |
| `GET /admin/users/:id` | — | `200 {user}` | `404 not_found` |
| `PATCH /admin/users/:id` | `{first_name,last_name,email?,national_id?,account_type}` | `200 {user}` | `400`, `404` |
| `POST /admin/users/:id/status` | `{status:"active"\|"disabled"}` | `200 {user}` | `400`, `404` |
| `POST /admin/users/:id/reset-password` | `{new_password}` | `204` | `400 weak_password`, `404` |
| `DELETE /admin/users/:id` | — | `204` | `404` |

Query-param parsing for `GET /admin/users`:
`first_name`, `last_name`, `phone`, `status` — trimmed; empty → omitted
filter. `created_from`, `created_to` — RFC3339 (`time.Parse(time.RFC3339, …)`);
also accept bare `2006-01-02`; bad value → `400`. `page`, `page_size` —
`strconv.Atoi`, default 1 / 20, non-positive falls back to default.

Persian error messages, matching the admin-auth handler tone:
- `409 phone_taken` → `"این شماره موبایل قبلاً ثبت شده است"`
- `400 validation_error` → `"ورودی نامعتبر است"`
- `400 weak_password` → `"رمز عبور باید حداقل ۸ نویسه و شامل حرف و رقم باشد"`
- `404 not_found` → `"کاربر یافت نشد"`

Map `user.ErrPhoneTaken` → 409, `user.ErrUserNotFound` → 404, everything
else validation-ish → 400.

### 7. Router — `internal/interface/http/router.go`

Inside the **existing** authenticated block

```go
p := r.Group("/admin")
p.Use(AdminAuth(tokens))
{
	p.GET("/me", a.me)
	...
```

add:

```go
	ua := &usersAdminHandler{svc: usersAdminSvc}
	p.GET("/users", ua.list)
	p.POST("/users", ua.create)
	p.GET("/users/:id", ua.get)
	p.PATCH("/users/:id", ua.update)
	p.POST("/users/:id/status", ua.setStatus)
	p.POST("/users/:id/reset-password", ua.resetPassword)
	p.DELETE("/users/:id", ua.remove)
```

`NewRouter` gains a `usersAdminSvc *usersadmin.Service` parameter (add it after
the admin params, before the client params, to keep the diff readable).

### 8. Wiring — `cmd/server/main.go`

```go
usersAdminSvc := usersadmin.NewService(
	repo.NewUserAdminRepo(sqlc.New(pool)),
	sms.FakeSender{Log: log.Default()},
)
router := httpiface.NewRouter(health.NewService(pool), authSvc, tokens, usersAdminSvc, clientSvc, clientTokens, cfg)
```

### 9. CORS — `internal/interface/http/middleware.go`

The list handler + friends use `PATCH` and `DELETE`. Update the one line:

```go
c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
```

### 10. Tests — `internal/application/usersadmin/service_test.go`

Match the style of `internal/application/adminauth/service_test.go` (stdlib
`testing`, hand-written in-memory fake, table loops, no framework). Cover:

- `Create` rejects a duplicate phone with `user.ErrPhoneTaken`
  (fake repo returns it on a seen phone).
- `Create` rejects a weak password and an invalid national id before hitting
  the repo.
- `Create` stores a bcrypt hash, not the plaintext, and calls `SMS.Send` once.
- `List` clamps `page`/`page_size` and computes `offset` correctly
  (e.g. page 3, size 10 → offset 20) and returns the repo's total.
- `SetStatus("disabled")` calls `RevokeAllSessions`; `SetStatus("bogus")`
  returns `user.ErrInvalidStatus` and touches nothing.
- `ResetPassword` rejects weak input, otherwise changes the stored hash and
  calls `RevokeAllSessions`.
- `Delete` calls `SoftDelete` then `RevokeAllSessions`; a later `Get` on the
  fake returns `user.ErrUserNotFound`.

Also add a couple of cases to `internal/domain/user/` for
`ValidateNationalID` (a known-good code, all-same-digits rejected, wrong
length rejected, bad checksum rejected) in a new
`internal/domain/user/national_id_test.go`.

### 11. Build / verify (Codex must run all of these)

```bash
cd api
sqlc generate                      # or: go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
go build ./...
go test ./...
# DB (from repo root): docker compose up -d postgres
migrate -path db/migration -database "$DATABASE_URL" up
go run ./cmd/seed
docker compose up -d --build api
curl -s localhost:8080/health
```

Then a real smoke test with the admin session (login `9370843199` /
`Amir@Pass1999` → bearer token) hitting `POST /admin/users`, `GET
/admin/users?...`, `GET/PATCH /admin/users/:id`, `POST .../status`, `POST
.../reset-password`, `DELETE`.

---

## Frontend (`./admin`)

Static-export SPA. All pages are full `"use client"` (repo precedent — no
Server Actions, no `revalidatePath` here). Reuse `Button` and `Field` exported
from `admin/app/login/page.tsx`, the `apiFetch` helper, and `useRequireAuth`.
Toaster is already global — use `toast.success|error|info`.

Design: existing tokens only (dark, `bg-brand-bg` / `bg-brand-surface`,
`border-brand-border`, `rounded-2xl` cards, `rounded-lg` controls, `h-11`
inputs, amber `bg-brand-accent` primary). No emoji as icons — small inline
SVG paths (Heroicons outline, `viewBox="0 0 24 24"`, `w-4 h-4`,
`stroke="currentColor"`). Transitions `transition-colors duration-200`.
`focus-visible:ring-2 focus-visible:ring-brand-accent/30` on every control,
never bare `outline-none`. Respect `prefers-reduced-motion` (only colour
transitions are used, so this is automatically fine). Table wrapper is
`overflow-x-auto`.

### New dependency

`admin/package.json`: add `"jalaali-js": "^1.2.6"` (pure ~4 KB
Jalali↔Gregorian conversion, zero deps). Run `npm install` in the worktree.
Rationale: Jalali dates are a product requirement; hand-rolling the calendar
math is a footgun, a full date-picker component is overkill for one filter.

### `admin/lib/users.ts` (new)

```ts
export type User = {
  id: number; first_name: string; last_name: string; email: string;
  phone_number: string; national_id: string | null;
  account_type: "individual" | "company";
  status: "active" | "disabled"; created_at: string;
};
export type ListParams = {
  first_name?: string; last_name?: string; phone?: string;
  status?: "active" | "disabled";
  created_from?: string; created_to?: string;
  page: number; page_size: number;
};
export type ListResult = { users: User[]; total: number; page: number; page_size: number };

export const listUsers   = (p: ListParams) => apiFetch(`/admin/users?${qs(p)}`, { auth: true }) as Promise<ListResult>;
export const getUser     = (id: number)    => apiFetch(`/admin/users/${id}`, { auth: true });
export const createUser  = (body: ...)     => apiFetch(`/admin/users`, { method: "POST", auth: true, body });
export const updateUser  = (id, body)      => apiFetch(`/admin/users/${id}`, { method: "PATCH", auth: true, body });
export const setStatus   = (id, status)    => apiFetch(`/admin/users/${id}/status`, { method: "POST", auth: true, body: { status } });
export const resetPassword = (id, new_password) => apiFetch(`/admin/users/${id}/reset-password`, { method: "POST", auth: true, body: { new_password } });
export const deleteUser  = (id: number)    => apiFetch(`/admin/users/${id}`, { method: "DELETE", auth: true });
```

`qs()` = local helper that drops `undefined` / `""` keys and
`encodeURIComponent`s the rest. `apiFetch` needs no change — it already
supports `method`, `auth`, `body` and returns `null` for `204`.

### `admin/lib/jalali.ts` (new, tiny)

```ts
import { toJalaali, toGregorian } from "jalaali-js";
export const FA_MONTHS = ["فروردین","اردیبهشت","خرداد","تیر","مرداد","شهریور","مهر","آبان","آذر","دی","بهمن","اسفند"];
// Gregorian ISO -> "۱۴۰۳/۰۵/۰۹" for display
export function isoToJalaliLabel(iso: string): string { ... }
// Jalali y/m/d -> [fromISO, toISO) UTC day bounds, for the created_at filter
export function jalaliDayRange(jy: number, jm: number, jd: number): { from: string; to: string } {
  const g = toGregorian(jy, jm, jd);
  const from = new Date(Date.UTC(g.gy, g.gm - 1, g.gd));
  const to = new Date(from.getTime() + 86400000);
  return { from: from.toISOString(), to: to.toISOString() };
}
```

`// ponytail: created_at filter matches a UTC day; small tz skew vs server
time is fine for a filter. Swap to a real calendar popover only if users
complain.`

### `admin/app/users/page.tsx` (new)

Full `"use client"`. Layout: reuse the page shell from `admin/app/page.tsx`
(centered `mx-auto max-w-*` with the `حساب` wordmark + logout header) — or
keep it plain with just a header row. `useRequireAuth()` guard → loading
screen identical to `settings/security/page.tsx`.

Structure:

1. **Header row** — `h1` «کاربران» on one side, a primary
   «＋ کاربر جدید» button (opens the modal) on the other.
2. **Table** in an `overflow-x-auto` `rounded-2xl border border-brand-border`
   card:
   - `thead` row 1: column titles — نام، نام خانوادگی، موبایل، نوع حساب،
     وضعیت، تاریخ ایجاد.
   - `thead` row 2 (the **filter row**, `bg-brand-bg`): under نام / نام
     خانوادگی / موبایل a compact `h-9` text `<input>`; under وضعیت a
     `<select>` (همه / فعال / غیرفعال); under تاریخ ایجاد the
     `<JalaliDateFilter>`; نوع حساب cell empty.
   - `tbody`: one `<tr>` per user, `hover:bg-brand-surface cursor-pointer`,
     click → `router.push(\`/users/${u.id}\`)`. Status cell renders a
     `<StatusBadge>`. Date cell shows `isoToJalaliLabel(u.created_at)`.
   - empty state: a single full-width row «کاربری یافت نشد».
   - while fetching: keep the old rows, drop opacity to `opacity-60`
     (avoids layout jump — `content-jumping`).
3. **Pagination footer** — right-aligned: «قبلی» button, «صفحه {page} از
   {totalPages}» label, «بعدی» button. `totalPages = Math.max(1,
   Math.ceil(total / page_size))`. Buttons `disabled` at the bounds.
4. **Create modal** — see below.

State + fetching:

```
const [filters, setFilters] = useState<Filters>({});   // first_name,last_name,phone,status,dateRange
const [page, setPage] = useState(1);
const debounced = useDebounced(filters, 300);          // tiny local hook
useEffect(() => { setPage(1); }, [debounced]);         // any filter change resets to page 1
useEffect(() => {
  let alive = true;
  setLoading(true);
  listUsers({ ...toParams(debounced), page, page_size: 20 })
    .then(r => { if (alive) { setData(r); } })
    .catch(e => toast.error(e instanceof Error ? e.message : "خطا"))
    .finally(() => { if (alive) setLoading(false); });
  return () => { alive = false; };
}, [debounced, page]);
```

`useDebounced` (local, ~6 lines):

```ts
function useDebounced<T>(v: T, ms: number): T {
  const [d, setD] = useState(v);
  useEffect(() => { const t = setTimeout(() => setD(v), ms); return () => clearTimeout(t); }, [v, ms]);
  return d;
}
```

**`<StatusBadge status>`** — `inline-flex items-center gap-1 rounded-full px-2
py-0.5 text-xs`; active → `bg-emerald-500/15 text-emerald-400` «فعال»;
disabled → `bg-brand-border text-brand-muted` «غیرفعال». A dot `<span
class="w-1.5 h-1.5 rounded-full bg-current">` (colour, not the only signal —
the text label carries it too).

**`<JalaliDateFilter value onChange>`** — three `<select>` (`h-9`): year
(`currentJalaliYear-6 … currentJalaliYear`), month (`FA_MONTHS`), day
(`1..31`), plus a «×» button when a full date is set. Only when all three are
chosen does it call `onChange(jalaliDayRange(y,m,d))`; «×» calls
`onChange(null)`. Invalid combos (e.g. 31 اسفند) — let the backend return
zero rows; don't over-validate.

**Create modal** — native `<dialog ref={dialogRef}>` opened with
`dialogRef.current.showModal()` and closed with `.close()`. `className` for
the box: `rounded-2xl border border-brand-border bg-brand-surface p-6 w-full
max-w-md text-brand-text backdrop:bg-black/50`. Form fields (reuse `Field`
where the shape fits, else same input classes):
first_name, last_name, phone_number (`inputMode="numeric"`), email
(`type="email"`, not required), national_id (not required), account_type
(`<select>`: حقیقی / حقوقی → `individual` / `company`), password
(`type="password"`). Submit handler:

```
try {
  await createUser(body);
  toast.success("کاربر ساخته شد");
  dialogRef.current?.close();
  // refetch current list
} catch (e) {
  toast.error(e instanceof Error ? e.message : "خطا");   // 409 → "این شماره موبایل قبلاً ثبت شده است"
}
```

Keep the modal open on error. Reset the form on successful close. `<Button
loading>` disables during submit (`loading-buttons`).

### `admin/app/users/[id]/page.tsx` (new)

Full `"use client"`. `const { id } = useParams<{ id: string }>()`.
`useRequireAuth()` guard. On mount `getUser(Number(id))`; on failure
`toast.error("کاربر یافت نشد")` + `router.replace("/users")`.

> Note: `output: "export"` + a dynamic route needs
> `export function generateStaticParams() { return []; }` in this file (or a
> route-segment config) so `next build` doesn't fail on the dynamic segment.
> Add it and confirm the export build passes; if Next 16 still refuses a
> param-less dynamic export, fall back to a query param
> (`/users?id=`) — but try `generateStaticParams` returning `[]` first.

Layout — a `mx-auto max-w-2xl p-6 sm:p-12` column:

1. **Back link** «→ بازگشت به فهرست» to `/users`.
2. **Profile card** (`rounded-2xl border bg-brand-surface p-6`): full name as
   the heading, `<StatusBadge>` beside it, then a definition list — موبایل،
   ایمیل، کد ملی، نوع حساب، تاریخ ایجاد (Jalali label). موبایل is read-only
   with a hint «شماره موبایل قابل تغییر نیست».
3. **Edit form** (same card or a second one): first_name, last_name, email,
   national_id, account_type. «ذخیره تغییرات» → `updateUser(id, body)` →
   `toast.success("ذخیره شد")` + refresh local state.
4. **Actions** — a card titled «عملیات» with clearly separated rows:
   - **Enable/disable**: one toggle button — label «غیرفعال‌سازی حساب» when
     active (amber/danger-ish styling), «فعال‌سازی حساب» when disabled →
     `setStatus(id, next)` → toast + refresh.
   - **Reset password**: a small inline form — one `type="password"` field
     «رمز عبور جدید» + «تغییر رمز عبور» button → `resetPassword(id, value)` →
     `toast.success("رمز عبور تغییر کرد")`, clear the field.
   - **Delete**: a `text-red-400` «حذف کاربر» button, gated by
     `if (!window.confirm("این کاربر حذف شود؟ این کار قابل بازگشت نیست."))
     return;` → `deleteUser(id)` → `toast.success("کاربر حذف شد")` +
     `router.push("/users")`.

Group visually: profile + edit at the top, a divider, then the «عملیات» card
so destructive controls are away from the edit fields.

### `admin/app/page.tsx` (1-line-ish edit)

Add a second link/button next to «تنظیمات امنیتی», pointing to `/users`,
same styling: «مدیریت کاربران».

### Frontend verify (Codex must run)

```bash
cd admin
npm install
npm run build      # static export to out/ — MUST be 0 errors
npm run dev        # :3010 — click through: list, filter (watch the network
                   # tab re-fetch), create via modal, open a detail page,
                   # disable, reset password, delete
```

Set `PYTHONIOENCODING=utf-8` if any ui-ux-pro-max script is invoked on this
Windows box.

---

## Out of scope / deliberately skipped

- Bulk select / bulk actions (ui-ux severity "Low" — add when a real need
  appears).
- Freeing a soft-deleted user's phone number for reuse (hard `UNIQUE` keeps
  it reserved; revisit if the product asks).
- `national_id` uniqueness (only phone was mandated unique).
- URL-syncing the filter state (nice for reload/share; add later if wanted).
- A real calendar-popover date picker (three selects cover the one filter).
- Real SMS (the fake stub + TODO already documents sms.ir).
- Client-side enforcement of `status='disabled'` at login — that belongs to
  `clientauth`; this task only sets the flag and revokes sessions.

## Acceptance criteria

1. `go build ./...` and `go test ./...` clean in `./api`; `npm run build`
   clean in `./admin`.
2. Migration `000003` applies and rolls back cleanly.
3. `POST /admin/users` with a duplicate phone → `409` +
   `"این شماره موبایل قبلاً ثبت شده است"`; the row is not created.
4. `GET /admin/users` honours every filter (`first_name`, `last_name`,
   `phone` substring; `status`; `created_at` day range) and paginates
   (`total`, `page`, `page_size` in the response).
5. The users page re-fetches from the API ~300 ms after the last keystroke /
   on every select or date change, and resets to page 1 on a filter change.
6. Create modal is a native `<dialog>`; on success the list refreshes and a
   success toast shows; on `409` the modal stays open with an error toast.
7. `/users/[id]` shows the detail, edits save, the disable toggle flips
   `status` and revokes the user's refresh tokens, reset-password enforces
   the strength rule and revokes tokens, delete is `confirm()`-gated, sets
   `deleted_at`, and returns to `/users` where the row is gone.
8. All feedback is via `toast.*` (no inline red `<p>` message state).
9. No emoji icons; every control has a visible focus ring; the table scrolls
   horizontally inside its own container on narrow viewports.
