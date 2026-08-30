# Plan — Businesses & Memberships (feat/businesses)

Multi-business support: a user owns 1..N businesses, invites others as members,
and works inside one business at a time. Admin panel manages all businesses.

Branch `feat/businesses`, worktree `C:\Users\root\Desktop\hesab-businesses`.
Base is `main` at the merge of `feat/users-admin` (users table already has
`national_id`, `account_type`, `status`, `deleted_at`; `/admin/users*` exists).

Run **ui-ux-pro-max** (scoped `admin:` / `client:` skill) for every frontend
page. Persian RTL, Vazirmatn, brand tokens, **toasts** (`sonner`) for every
feedback message — no inline `<p>` error state. Prefix any Python-backed
tooling call with `export PYTHONIOENCODING=utf-8`.

---

## 1. Decisions (locked — do not re-litigate)

| Topic | Decision |
|---|---|
| Model | `businesses.owner_user_id` → `users.id` (1:N). Business fields for now: `name` only. |
| Roles | `owner`, `admin`, `accountant`, `viewer`. |
| Membership | `business_members(business_id, user_id, role)`; owner also gets a row with `role='owner'`. |
| Client invite | by phone → must resolve to a registered (non-deleted) user → creates a **pending invite**; invitee accepts/rejects on next login. Unknown phone → toast "کاربر باید ابتدا در سامانه ثبت‌نام کند". |
| Admin invite (on `/users/:id` and `/businesses/:id`) | pick user + role → **immediate** membership, no accept step. |
| Manage-members rights (client) | only `owner` + `admin` can invite / remove / change role. `accountant` / `viewer` read-only. |
| Delete business | soft delete (`deleted_at`). Client: `owner` only. Admin: always. |
| `/users/:id` business list | shows **owned + joined**. Owned = full CRUD via admin business endpoints. Joined = show role + "remove from business". |
| Client post-login | land on `/select-business`. 0 businesses & 0 invites → empty state + create CTA. Exactly 1 business & 0 invites → auto-redirect into it. Else → picker. |
| Client structure | business-scoped routes: `/businesses/:id/dashboard`, `/businesses/:id/members`, `/businesses/:id/settings`. |
| Header switcher | `<select>` of the user's businesses; on change navigate to the **same sub-path** under the new id (`/businesses/A/members` → `/businesses/B/members`). **Not** persisted to localStorage/anywhere — state lives in the URL only. |
| Client sub-pages this task | `dashboard` (placeholder), `members`, `settings`, plus pending-invites inbox on `/select-business`. |
| Seed | seeder gives user `9120000000` one business `کسب‌وکار نمونه` (owner + owner member row), idempotent. |
| Migration | `000004_businesses`. |
| Error envelope | `{"error":{"code":"...","message":"<Persian>"}}` (matches existing handlers). Success = plain JSON like `usersadmin`. |

---

## 2. Database — `api/db/migration/000004_businesses.{up,down}.sql`

### up
```sql
CREATE TABLE businesses (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          TEXT   NOT NULL,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
CREATE INDEX businesses_owner_idx      ON businesses (owner_user_id) WHERE deleted_at IS NULL;
CREATE INDEX businesses_created_at_idx ON businesses (created_at DESC);

CREATE TABLE business_members (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    role        TEXT   NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, user_id)
);
CREATE INDEX business_members_user_idx ON business_members (user_id);

CREATE TABLE business_invites (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id  BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users(id)      ON DELETE CASCADE,  -- invitee
    role         TEXT   NOT NULL,
    invited_by   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status       TEXT   NOT NULL DEFAULT 'pending',  -- pending | accepted | rejected | cancelled
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX business_invites_pending_uniq
    ON business_invites (business_id, user_id) WHERE status = 'pending';
```

### down
```sql
DROP TABLE IF EXISTS business_invites;
DROP TABLE IF EXISTS business_members;
DROP TABLE IF EXISTS businesses;
```

No changes to `users`. New tables are purely additive → `clientauth` /
`usersadmin` keep compiling.

---

## 3. Domain — `api/internal/domain/business/business.go`

```go
package business

const (
    RoleOwner      = "owner"
    RoleAdmin      = "admin"
    RoleAccountant = "accountant"
    RoleViewer     = "viewer"
)

func ValidRole(s string) bool        // any of the four
func AssignableRole(s string) bool   // admin | accountant | viewer  (cannot invite/assign "owner")
func CanManageMembers(role string) bool // owner || admin
```

Errors (package vars):
`ErrNotFound`, `ErrNameRequired`, `ErrInvalidRole`, `ErrNotMember`,
`ErrForbidden`, `ErrAlreadyMember`, `ErrInvitePending`, `ErrInviteNotFound`,
`ErrInviteeNotRegistered`, `ErrCannotTargetOwner`, `ErrOwnerCannotLeave`.

Types:
```go
type Business struct { ID, OwnerUserID int64; Name string; CreatedAt time.Time; DeletedAt *time.Time }
type Member  struct { UserID int64; Role, FirstName, LastName, PhoneNumber string; CreatedAt time.Time }
type Invite  struct { ID, BusinessID int64; BusinessName, Role, Status, InvitedByName string; CreatedAt time.Time }
```
`Name` cleaning: `strings.TrimSpace`; empty → `ErrNameRequired`. (No length cap now.)

---

## 4. sqlc queries

`api/query/businesses.sql` (client-facing, no prefix) and
`api/query/businesses_admin.sql` (`Admin`-prefixed) — keep the split style used
by `client_auth.sql` vs `users_admin.sql`. All list/detail queries filter
`b.deleted_at IS NULL`.

### `businesses.sql`
- `CreateBusiness :one` — insert `(name, owner_user_id)` returning row.
- `AddMember :one` — insert `(business_id, user_id, role)` returning row (used for owner row on create + accepted invites + admin immediate-add).
- `GetBusiness :one` — `SELECT * FROM businesses WHERE id=$1 AND deleted_at IS NULL`.
- `GetMemberRole :one` — `SELECT role FROM business_members WHERE business_id=$1 AND user_id=$2`.
- `ListUserBusinesses :many` — businesses the user owns or is a member of, with the caller's role:
  ```sql
  SELECT b.id, b.name, b.owner_user_id, b.created_at, m.role
  FROM businesses b
  JOIN business_members m ON m.business_id = b.id AND m.user_id = $1
  WHERE b.deleted_at IS NULL
  ORDER BY b.created_at;
  ```
  (owner always has a member row, so this one join covers both.)
- `ListMembers :many` — join `users`, return `user_id, role, first_name, last_name, phone_number, created_at` for a business, `ORDER BY created_at`.
- `RenameBusiness :one` — `UPDATE businesses SET name=$2 WHERE id=$1 AND deleted_at IS NULL RETURNING *`.
- `SoftDeleteBusiness :exec` — `UPDATE businesses SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL`.
- `UpdateMemberRole :one` — `UPDATE business_members SET role=$3 WHERE business_id=$1 AND user_id=$2 RETURNING *`.
- `RemoveMember :exec` — `DELETE FROM business_members WHERE business_id=$1 AND user_id=$2`.
- `GetActiveUserByPhone :one` — `SELECT * FROM users WHERE phone_number=$1 AND deleted_at IS NULL` (invite lookup; do **not** reuse `GetUserByPhone`, it ignores `deleted_at`).
- `CreateInvite :one` — insert `(business_id, user_id, role, invited_by)` returning row.
- `ListPendingInvitesForUser :many` — invites where `user_id=$1 AND status='pending'`, join `businesses` (name) and `users` inviter (name).
- `ListPendingInvitesForBusiness :many` — `business_id=$1 AND status='pending'`, join invitee `users`.
- `GetInvite :one` — by id.
- `SetInviteStatus :exec` — `UPDATE business_invites SET status=$2, responded_at=now() WHERE id=$1`.

### `businesses_admin.sql`
- `AdminListBusinesses :many` — filter `name ILIKE '%'||narg||'%'`, `deleted_at IS NULL`, `ORDER BY created_at DESC`, `LIMIT lim OFFSET off`; select `b.*`, owner `first_name,last_name,phone_number` (join users), and `member_count` (`(SELECT count(*) FROM business_members WHERE business_id=b.id)`).
- `AdminCountBusinesses :one` — same filter, `count(*)`.
- `AdminGetBusiness :one` — `b.*` + owner fields.
- `AdminListOwnedBusinesses :many` — `WHERE owner_user_id=$1 AND deleted_at IS NULL`, + member_count. (for `/users/:id`)
- `AdminListJoinedBusinesses :many` — businesses where user has a member row with `role <> 'owner'`, `deleted_at IS NULL`, + owner name + the user's role. (for `/users/:id`)

Reuse existing `GetUserByID` (client_auth.sql) where an owner-id needs validating.

Run `sqlc generate` (config `api/sqlc.yaml`, out `internal/infrastructure/db/sqlc`). Commit generated files. Expect CRLF churn — `git add` may warn LF→CRLF; that's fine.

---

## 5. Application layer

### `api/internal/application/business/service.go` (client use-cases)

`Repository` interface wrapping the `businesses.sql` queries + a
`GetUserByID`. `Service` methods (all take `ctx` + acting `userID`):

- `List(ctx, userID) ([]BusinessWithRole, error)`
- `Create(ctx, userID, name) (Business, error)` — trim/validate name; `CreateBusiness`; then `AddMember(bizID, userID, RoleOwner)`; return.
- `Get(ctx, userID, bizID) (Business, callerRole string, error)` — `GetMemberRole` → `ErrNotMember` (404 to client) if none; else `GetBusiness`.
- `Rename(ctx, userID, bizID, name)` — require `CanManageMembers(role)` else `ErrForbidden`; validate name; `RenameBusiness`.
- `Delete(ctx, userID, bizID)` — require `role == RoleOwner` else `ErrForbidden`; `SoftDeleteBusiness`. (Invites/members cascade stay; soft-deleted business is filtered everywhere.)
- `Members(ctx, userID, bizID) ([]Member, callerRole, error)` — require membership.
- `Invite(ctx, userID, bizID, phone, role)` — require `CanManageMembers`; `AssignableRole(role)` else `ErrInvalidRole`; normalize phone via `user.NormalizePhone`; `GetActiveUserByPhone` → `ErrInviteeNotRegistered` (404) if missing; if that user already has a member row → `ErrAlreadyMember` (409); if a pending invite exists → `ErrInvitePending` (409); else `CreateInvite(status=pending, invited_by=userID)`.
- `CancelInvite(ctx, userID, bizID, inviteID)` — require `CanManageMembers`; invite must belong to biz & be pending; `SetInviteStatus('cancelled')`.
- `ChangeRole(ctx, userID, bizID, targetUserID, role)` — require `CanManageMembers`; `AssignableRole(role)`; target's current role must not be `owner` (`ErrCannotTargetOwner`); `UpdateMemberRole`.
- `RemoveMember(ctx, userID, bizID, targetUserID)` — if `targetUserID == userID`: self-leave, allowed for any non-owner role; owner → `ErrOwnerCannotLeave`. Else: require `CanManageMembers`; target role `owner` → `ErrCannotTargetOwner`; `RemoveMember`.
- `PendingInvites(ctx, userID) ([]Invite, error)` — `ListPendingInvitesForUser`.
- `OutgoingInvites(ctx, userID, bizID) ([]Invite, error)` — require `CanManageMembers`; `ListPendingInvitesForBusiness`.
- `AcceptInvite(ctx, userID, inviteID)` — `GetInvite`; must be `user_id == userID` & `status=='pending'` (else `ErrInviteNotFound`); if already a member just `SetInviteStatus('accepted')`; else `AddMember(bizID, userID, invite.role)` then `SetInviteStatus('accepted')`.
- `RejectInvite(ctx, userID, inviteID)` — same ownership check; `SetInviteStatus('rejected')`.

### `api/internal/application/businessadmin/service.go` (admin use-cases)

`Repository` wrapping `businesses_admin.sql` + `businesses.sql`'s member
mutators + `GetActiveUserByPhone` + `GetUserByID`. Methods:

- `List(ctx, nameFilter, page, pageSize) (rows, total, error)` — clamp page/size like `usersadmin` (default 20, max 100).
- `Create(ctx, ownerUserID, name) (Business, error)` — validate owner exists (`GetUserByID` → `user.ErrUserNotFound`); validate name; `CreateBusiness`; `AddMember(bizID, ownerUserID, RoleOwner)`.
- `Get(ctx, bizID) (Business, owner, []Member, error)`.
- `Rename(ctx, bizID, name) (Business, error)`.
- `Delete(ctx, bizID) error` — soft delete, no role gate (super-admin).
- `AddMemberByPhone(ctx, bizID, phone, role) (Member, error)` — `AssignableRole`; `GetActiveUserByPhone` → `ErrInviteeNotRegistered` (404); already member → `ErrAlreadyMember` (409); else `AddMember` immediately (no invite row).
- `ChangeRole(ctx, bizID, targetUserID, role)` / `RemoveMember(ctx, bizID, targetUserID)` — same owner-guard as client side; no acting-role gate.
- `UserBusinesses(ctx, targetUserID) (owned []OwnedRow, joined []JoinedRow, error)`.

---

## 6. HTTP layer

### `api/internal/interface/http/businesses_admin_handler.go`
`type businessesAdminHandler struct{ svc *businessadmin.Service }`.
Reuse `parseID`, `errJSON`, `optional` from `users_admin_handler.go`.
For a second path param use `strconv.ParseInt(c.Param("userId"), ...)`.

Routes (register in `router.go` **inside the existing `p := r.Group("/admin")`
/ `p.Use(AdminAuth(tokens))` block**):
```
GET    /admin/businesses                         list  (?name=&page=&page_size=)
POST   /admin/businesses                         create {name, owner_user_id}
GET    /admin/businesses/:id                     detail (business + members)
PATCH  /admin/businesses/:id                     rename {name}
DELETE /admin/businesses/:id                     soft delete
POST   /admin/businesses/:id/members             add {phone_number, role}   (immediate)
PATCH  /admin/businesses/:id/members/:userId     change role {role}
DELETE /admin/businesses/:id/members/:userId     remove
GET    /admin/users/:id/businesses               {owned:[...], joined:[...]}
```
Note the last route shares the `:id` param name with the existing
`/admin/users/:id` routes — that's fine, same group, add it next to them.

### `api/internal/interface/http/businesses_handler.go`
`type businessesHandler struct{ svc *business.Service }`.
Acting user id from context: `c.GetInt64("userID")` (set by `ClientAuth`).

Routes (register **inside the existing `cp := r.Group("/client")` /
`cp.Use(ClientAuth(clientTokens))` block**):
```
GET    /client/businesses                              list (caller's businesses + role)
POST   /client/businesses                              create {name}
GET    /client/businesses/:id                          detail {business, role}
PATCH  /client/businesses/:id                          rename {name}
DELETE /client/businesses/:id                          soft delete (owner only)
GET    /client/businesses/:id/members                  {members, role}
POST   /client/businesses/:id/members                  invite {phone_number, role}  -> pending
DELETE /client/businesses/:id/members/:userId          remove / leave
PATCH  /client/businesses/:id/members/:userId          change role {role}
GET    /client/businesses/:id/invites                  outgoing pending invites (owner/admin)
DELETE /client/businesses/:id/invites/:inviteId        cancel outgoing invite
GET    /client/invites                                 my pending invites
POST   /client/invites/:id/accept
POST   /client/invites/:id/reject
```

### Error → HTTP mapping (both handlers)
| error | status | code |
|---|---|---|
| `business.ErrNotFound`, `business.ErrNotMember` (client GET) | 404 | `not_found` |
| `business.ErrForbidden` | 403 | `forbidden` |
| `business.ErrNameRequired` | 400 | `validation_error` |
| `business.ErrInvalidRole` | 400 | `invalid_role` |
| `business.ErrInviteeNotRegistered` | 404 | `user_not_registered` (message "کاربر با این شماره در سامانه ثبت‌نام نکرده است") |
| `business.ErrAlreadyMember` | 409 | `already_member` |
| `business.ErrInvitePending` | 409 | `invite_pending` |
| `business.ErrInviteNotFound` | 404 | `not_found` |
| `business.ErrCannotTargetOwner` | 409 | `owner_immutable` |
| `business.ErrOwnerCannotLeave` | 409 | `owner_cannot_leave` |
| `user.ErrUserNotFound` (admin create bad owner) | 404 | `not_found` |
| default | 400 | `validation_error` |

### JSON shapes
```
business:  {id, name, owner_user_id, created_at}
member:    {user_id, first_name, last_name, phone_number, role, created_at}
invite (incoming): {id, business_id, business_name, role, invited_by_name, created_at}
invite (outgoing): {id, business_id, user_id, first_name, last_name, phone_number, role, created_at}
admin list row: {id, name, member_count, created_at, owner:{id, first_name, last_name, phone_number}}
admin detail: {business:{...}, owner:{...}, members:[member...]}
admin user-businesses: {owned:[{id,name,member_count,created_at}], joined:[{id,name,role,owner_name,created_at}]}
client list row: {id, name, role, owner_user_id, created_at}
```
Envelopes: `{"businesses":[...]}`, `{"business":{...}}`, `{"members":[...],"role":"..."}`, `{"invites":[...]}`, list adds `total,page,page_size`.

### `router.go` + `main.go` wiring
- `NewRouter(...)` gains two params: `businessAdminSvc *businessadmin.Service, businessSvc *business.Service`. Add after `usersAdminSvc` / before `clientSvc` to keep admin things together; update the call in `main.go`.
- `main.go`: build
  ```go
  q := sqlc.New(pool)
  businessSvc := business.NewService(repo.NewBusinessRepo(q))
  businessAdminSvc := businessadmin.NewService(repo.NewBusinessAdminRepo(q))
  ```
  (existing code calls `sqlc.New(pool)` per service; a shared `q` is fine too — match local style, don't refactor the others.)

### Repos
`api/internal/infrastructure/repo/business_repo.go` and
`business_admin_repo.go` — same shape as `user_admin_repo.go`: hold `*sqlc.Queries`,
map `sqlc` rows → `business.*` structs, translate `pgx.ErrNoRows` →
`business.ErrNotFound` / `business.ErrNotMember` as appropriate, `23505` on
`business_invites` pending-unique → `business.ErrInvitePending`, `23505` on
`business_members` → `business.ErrAlreadyMember`.

---

## 7. Seeder — `api/cmd/seed/main.go`

After the existing user upsert (which already returns `id` into `id`), add:
```go
var bizID int64
e = pool.QueryRow(ctx, `
  INSERT INTO businesses (name, owner_user_id)
  SELECT $1, $2
  WHERE NOT EXISTS (SELECT 1 FROM businesses WHERE owner_user_id = $2 AND name = $1 AND deleted_at IS NULL)
  RETURNING id`, "کسب‌وکار نمونه", id).Scan(&bizID)
if e == nil {
  _, e = pool.Exec(ctx, `INSERT INTO business_members (business_id, user_id, role)
    VALUES ($1, $2, 'owner') ON CONFLICT (business_id, user_id) DO NOTHING`, bizID, id)
}
if e != nil && e != pgx.ErrNoRows { log.Fatal(e) }
log.Printf("seeded business id=%d owner=%d", bizID, id)
```
`pgx.ErrNoRows` when the business already exists — treat as ok.

---

## 8. Frontend — admin (`./admin`)

### `admin/lib/businesses.ts` (mirror `lib/users.ts`)
Types + fetchers via `apiFetch(..., {auth:true})`:
`listBusinesses({name,page,page_size})`, `createBusiness({name,owner_user_id})`,
`getBusiness(id)`, `renameBusiness(id,{name})`, `deleteBusiness(id)`,
`addBusinessMember(id,{phone_number,role})`,
`changeBusinessMemberRole(id,userId,{role})`,
`removeBusinessMember(id,userId)`, `getUserBusinesses(userId)`.
Role labels map: `owner`→«مالک», `admin`→«مدیر», `accountant`→«حسابدار», `viewer`→«ناظر».

### `admin/app/businesses/page.tsx` — list
Clone the structure of `admin/app/users/page.tsx`:
- header «کسب‌وکارها» + «کسب‌وکار جدید» button opening a native `<dialog>`.
- one debounced text filter: name. Server pagination (page size 20), «قبلی/بعدی».
- columns: نام | مالک (link `/users/{owner.id}`, shows name + `dir="ltr"` phone) | تعداد اعضا | تاریخ ایجاد. Row click → `/businesses/{id}`.
- Create modal: `نام کسب‌وکار` text input + **owner picker**. Owner picker =
  a phone text input; on submit call `listUsers({phone, page:1, page_size:5})`,
  if exactly one match use it, if several show a small radio list to pick, if
  none `toast.error("کاربری با این شماره یافت نشد")`. Submit → `createBusiness`.
- `isoToJalaliLabel` for dates, Persian digits via existing helpers.

### `admin/app/businesses/[id]/page.tsx` + `Detail.tsx`
Same static-export pattern as `admin/app/users/[id]/`:
```tsx
// page.tsx
import Detail from "./Detail";
export const dynamicParams = false;
export function generateStaticParams() { return [{ id: "_" }]; }
export default function Page() { return <Detail />; }
```
`Detail.tsx` (`"use client"`, `useParams`, `useRequireAuth`):
- back link → `/businesses`.
- header: business name + owner (link to `/users/{ownerId}`) + created date.
- rename form (`renameBusiness`) → toast.
- members table: نام | موبایل (`dir="ltr"`) | نقش | تاریخ عضویت | (actions).
  - role cell: `<select>` (owner/admin/accountant/viewer) — disabled on the owner row; change → `changeBusinessMemberRole` → toast.
  - remove button — hidden on owner row; `confirm()` → `removeBusinessMember` → toast + refresh.
- "add member" row: phone input + role `<select>` (admin/accountant/viewer) → `addBusinessMember` → toast + refresh. 404 `user_not_registered` → show the server message via `toast.error`.
- danger: «حذف کسب‌وکار» — `window.confirm("این کسب‌وکار حذف شود؟")` → `deleteBusiness` → toast + `router.push("/businesses")`.

### `admin/app/users/[id]/Detail.tsx` — add a "کسب‌وکارها" section
Below the existing «عملیات» section. On mount also call `getUserBusinesses(id)`.
- «کسب‌وکارهای تحت مالکیت» — list: name (link `/businesses/{bid}`) + member count + created. Inline «افزودن کسب‌وکار» form: single name input → `createBusiness({name, owner_user_id:id})` → toast + refresh list.
- «عضویت‌ها» — list: name (link `/businesses/{bid}`) + role label + owner name. Each row a «حذف از کسب‌وکار» button → `confirm()` → `removeBusinessMember(bid, id)` → toast + refresh.

### `admin/serve.json`
Add the businesses rewrite:
```json
{ "rewrites": [
  { "source": "/users/:id", "destination": "/users/_.html" },
  { "source": "/businesses/:id", "destination": "/businesses/_.html" }
] }
```

### nav
On `admin/app/page.tsx` add a «مدیریت کسب‌وکارها» link next to «مدیریت کاربران»
(`/businesses`). On `admin/app/users/page.tsx` and
`admin/app/businesses/page.tsx` headers, a small link across to the other list.

---

## 9. Frontend — client (`./client`) — business-scoped restructure

### `client/lib/businesses.ts`
Types + fetchers (all `{auth:true}`):
`listBusinesses()`, `createBusiness({name})`, `getBusiness(id)`,
`renameBusiness(id,{name})`, `deleteBusiness(id)`, `listMembers(id)`,
`inviteMember(id,{phone_number,role})`, `removeMember(id,userId)`,
`changeMemberRole(id,userId,{role})`, `listOutgoingInvites(id)`,
`cancelInvite(id,inviteId)`, `listMyInvites()`, `acceptInvite(inviteId)`,
`rejectInvite(inviteId)`.
Role label map same as admin. `canManage = (role) => role==="owner"||role==="admin"`.

### Routing / static-export pattern
Dynamic segment `businesses/[id]` — use a **server** `layout.tsx` that exports
the static-params boilerplate and renders a `"use client"` shell:
```tsx
// client/app/businesses/[id]/layout.tsx   (server component)
import Shell from "./Shell";
export const dynamicParams = false;
export function generateStaticParams() { return [{ id: "_" }]; }
export default function Layout({ children }: { children: React.ReactNode }) {
  return <Shell>{children}</Shell>;
}
```
Sub-pages (`dashboard/page.tsx`, `members/page.tsx`, `settings/page.tsx`) are
plain `"use client"` components, no per-page `generateStaticParams` needed —
they inherit the `[id]` static param from the layout. Static export emits
`out/businesses/_/dashboard.html`, `.../members.html`, `.../settings.html`.

### `client/serve.json` (new) + `client/package.json` start script
```json
{ "rewrites": [
  { "source": "/businesses/:id",            "destination": "/businesses/_/dashboard.html" },
  { "source": "/businesses/:id/dashboard",   "destination": "/businesses/_/dashboard.html" },
  { "source": "/businesses/:id/members",     "destination": "/businesses/_/members.html" },
  { "source": "/businesses/:id/settings",    "destination": "/businesses/_/settings.html" }
] }
```
Change `start` to `npx --yes serve out -l 3020 -c serve.json` (file sits in
`client/`, cwd is `client/` when npm runs it — use the bare name, not `../`).

### `client/app/businesses/[id]/Shell.tsx`  (`"use client"`)
- `useRequireAuth()`; `useParams<{id:string}>()`; `usePathname()`.
- On mount / id change: `getBusiness(id)` → on 403/404 `toast.error("دسترسی ندارید")` + `router.replace("/select-business")`. Keep `{business, role}` in state; also `listBusinesses()` for the switcher options.
- Header: brand «حساب» | business `<select>` (options from `listBusinesses`, value = current id) | nav links داشبورد/اعضا/تنظیمات (each `/businesses/{id}/<seg>`, active state from `usePathname`) | link «تنظیمات امنیتی» → `/settings/security` | «خروج» (logout: `apiFetch("/client/auth/logout",{method:"POST"})` + `clearSession()` + `router.replace("/login")`).
- Switcher `onChange`: compute the current sub-segment from `usePathname()`
  (`/businesses/{id}/(\w+)` → `seg`, default `dashboard`) and
  `router.push('/businesses/' + newId + '/' + seg)`. **No** localStorage / cookie / context — nothing persisted.
- Render `{children}` in a `<main>`.
- Provide `{business, role, reload}` to children via a React context
  (`BusinessContext`) exported from this file, so sub-pages don't refetch the
  business just for the role. (Members/settings still fetch their own lists.)

### `client/app/businesses/[id]/dashboard/page.tsx`  (`"use client"`)
Placeholder: card «داشبورد حسابداری» + «خوش آمدید» + business name from context.
(Move the body of the old `client/app/page.tsx` here, minus the header/logout —
those live in Shell now.)

### `client/app/businesses/[id]/members/page.tsx`  (`"use client"`)
- `listMembers(id)` → table نام | موبایل (`dir="ltr"`) | نقش | تاریخ عضویت | actions.
- If `canManage(role)`:
  - role `<select>` per row (admin/accountant/viewer), disabled on owner row and on the caller's own row; change → `changeMemberRole` → toast + refresh.
  - «حذف» per row (not owner row) → `confirm()` → `removeMember` → toast + refresh.
  - «دعوت عضو» form: phone input + role `<select>` (admin/accountant/viewer) → `inviteMember` → on success `toast.success("دعوت ارسال شد")`; 404 `user_not_registered` → `toast.error(<server message>)`; 409 → server message.
  - outgoing pending invites section: `listOutgoingInvites(id)` → list نام/موبایل + نقش + «لغو دعوت» (`cancelInvite`).
- else (accountant/viewer): table only, no controls.
- Self-leave: if caller role !== owner, a «خروج از این کسب‌وکار» button →
  `confirm()` → `removeMember(id, myUserId)` → `router.replace("/select-business")`.
  (Need caller's own user id — get it from `useRequireAuth().user.id`.)

### `client/app/businesses/[id]/settings/page.tsx`  (`"use client"`)
- rename form (only if `canManage(role)`) → `renameBusiness` → toast + context reload.
- if `role === "owner"`: «حذف کسب‌وکار» → `window.confirm("این کسب‌وکار حذف شود؟ قابل بازگشت نیست.")` → `deleteBusiness` → `router.replace("/select-business")`.
- if `role !== "owner"`: «خروج از کسب‌وکار» → same as members-page leave.

### `client/app/select-business/page.tsx`  (`"use client"`)  — picker + invites inbox
- `useRequireAuth()`. Then `Promise.all([listBusinesses(), listMyInvites()])`.
- **0 businesses & 0 invites** → empty state card: «شما هنوز کسب‌وکاری ندارید» + button «ساخت کسب‌وکار» → `router.push("/businesses/new")`.
- **exactly 1 business & 0 invites** → `router.replace('/businesses/'+b.id+'/dashboard')`.
- otherwise → 
  - list businesses as cards/rows (name + your-role badge) → click → `/businesses/{id}/dashboard`.
  - «ساخت کسب‌وکار جدید» button → `/businesses/new`.
  - if invites: «دعوت‌ها» section — each: business name + role + inviter →
    «پذیرفتن» (`acceptInvite` → refetch; if now exactly 1 business & 0 invites, go into it) / «رد کردن» (`rejectInvite` → refetch). Toasts on both.

### `client/app/businesses/new/page.tsx`  (`"use client"`)
Simple centered card: one name input + «ساخت» → `createBusiness({name})` →
`toast.success` → `router.replace('/businesses/'+res.business.id+'/dashboard')`.
Back link → `/select-business`.

### `client/app/page.tsx`
Replace body with: `useRequireAuth()` then `router.replace("/select-business")`
(show the standard loading `<main>` while deciding). Keeps `/` working as an
entry point.

### `client/app/login/page.tsx` + `client/app/login/2fa/page.tsx`
Change the post-login success redirect from `r.replace("/")` to
`r.replace("/select-business")`. (`/` also redirects there, but do it directly
to avoid a flash.)

### `client/app/settings/security/page.tsx`
Leave as-is (global, not business-scoped). If it has a back link to `/`, that
still lands on the picker — acceptable.

---

## 10. Acceptance criteria

**Backend**
- `cd api && go build ./... && go vet ./... && go test ./...` all clean.
- `sqlc generate` runs clean; generated files committed.
- Against host DB `hesab-postgres-1` on `:5432`:
  `migrate -path db/migration -database "$DATABASE_URL" up` applies `000004`;
  `go run ./cmd/seed` prints a `seeded business id=…` line and is idempotent on
  a second run.
- Manual curl (server on `:8081` per api/CLAUDE.md), client token for
  `9120000000`:
  - `GET /client/businesses` → the seeded `کسب‌وکار نمونه`, role `owner`.
  - `POST /client/businesses {name:"دوم"}` → 201; now list has 2.
  - `POST /client/businesses/{id}/members {phone_number:"9120000000",role:"admin"}`
    while already a member → 409 `already_member`.
  - invite an unknown phone → 404 `user_not_registered`.
  - create a 2nd user via `/admin/users`, invite by phone → 200; that user
    `GET /client/invites` → sees it; `POST /client/invites/{id}/accept` → then
    appears in their `GET /client/businesses` with the invited role.
  - `viewer` calling `POST .../members` → 403 `forbidden`.
  - `DELETE /client/businesses/{id}` as non-owner → 403; as owner → 204 and it
    disappears from every list.
- Admin: `GET /admin/businesses?name=نمونه` paginates; `POST /admin/businesses`
  with `owner_user_id` creates + owner member row; `/admin/users/{id}/businesses`
  returns owned + joined.

**Frontend**
- `cd admin && npm run build` and `cd client && npm run build` → 0 errors,
  static export completes.
- Admin `/businesses`, `/businesses/[id]`, and the new section on `/users/[id]`
  render RTL with brand tokens, all feedback via toasts.
- Client: login → `/select-business`; the three states (0 / exactly-1 / many)
  behave per the table in §1; header switcher changes the URL id while keeping
  the sub-path and persists nothing; invite-by-phone shows pending + toast;
  unknown phone → toast error; owner/admin gating hides controls for
  accountant/viewer; owner-only delete in settings.

---

## 11. Out of scope (do NOT build)

- Business fields beyond `name` (coming later).
- Owner transfer; owner leaving a business.
- Real accounting data / dashboard widgets.
- Email/SMS notification of invites (invitee just sees them on next login).
- Per-role permission enforcement on future accounting endpoints.
- Editing `client/app/settings/security` beyond leaving it working.
- CI.

---

## 12. Notes / gotchas (from repo CLAUDE.md files)

- No `make`. Migrate CLI at `~/go/bin/migrate`. Seed: `go run ./cmd/seed`.
- Host DB is `hesab-postgres-1` on `:5432`; `docker compose up` **from the
  worktree** creates a separate empty stack — don't use it. Server on `:8080`
  is the old container; run a fresh one on `:8081` for manual checks.
- `output: "export"` ignores `next.config` `rewrites`; dynamic routes need
  `dynamicParams=false` + `generateStaticParams` returning `[{id:"_"}]`.
- Next 16 `next dev` rewrites `tsconfig.json` / `next-env.d.ts` / `AGENTS.md` —
  commit those, don't fight them. IDE TS/`gopls` diagnostics lag; `next build`
  and `go build` are the source of truth.
- CRLF: freshly generated files show as modified with empty `git diff` —
  `git checkout -- <path>` to clear, don't commit noise.
- All user-facing messages → `toast.*` (sonner), never inline `<p>`.
- Run the `admin:` / `client:` `ui-ux-pro-max` skill for the frontend work;
  prefix Python tooling with `export PYTHONIOENCODING=utf-8`.
