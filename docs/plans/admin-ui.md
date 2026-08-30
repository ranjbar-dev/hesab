# Admin UI overhaul + prod routing fix + avatar upload

Implement everything below. Build both `admin` and `client` with `npm run
build` when done (static export must succeed with 0 errors) and `go build
./...` in `api`. Do not stop early — every section is required.

## 0. Ground rules

- `admin` = dark theme (`--color-brand-*` in `admin/app/globals.css`),
  `client` = light theme (same token names, different values in
  `client/app/globals.css`). Reuse those tokens, don't hardcode colors.
- Both apps are `output: "export"` static SPAs (no server, no route
  handlers, no `next/image` optimization). Every new interactive file needs
  `"use client"`.
- RTL (`dir="rtl"` on `<html>`) in both apps. "Right" in this doc means the
  RTL leading edge (visually right in a browser).
- Persian UI strings throughout, matching the existing tone (see current
  pages for examples).

---

## 1. Nginx deep-link fix (prod bug)

`https://admin.ranjbar.dev/users/1` and `/businesses/1` currently render the
dashboard because nginx's catch-all `try_files $uri $uri.html $uri/
/index.html` never finds `/users/1.html` and falls through to the SPA shell
at `/`, which is the dashboard route. The static export actually emits
`out/users/_.html` / `out/businesses/_.html` (see `admin/serve.json` — the
dev/local `serve` rewrites already do this correctly). Client is the same
problem for `out/businesses/_/{dashboard,members,settings}.html` (see
`client/serve.json`).

Edit `deploy/nginx/admin.ranjbar.dev.conf`: add, **before** the existing
`location /` block:

```nginx
location ~ ^/users/[^/]+/?$ {
    try_files /users/_.html =404;
}
location ~ ^/businesses/[^/]+/?$ {
    try_files /businesses/_.html =404;
}
```

Edit `deploy/nginx/ranjbar.dev.conf`: add, **before** the existing
`location /` block, mirroring `client/serve.json` exactly:

```nginx
location ~ ^/businesses/[^/]+/?$ {
    try_files /businesses/_/dashboard.html =404;
}
location ~ ^/businesses/[^/]+/dashboard/?$ {
    try_files /businesses/_/dashboard.html =404;
}
location ~ ^/businesses/[^/]+/members/?$ {
    try_files /businesses/_/members.html =404;
}
location ~ ^/businesses/[^/]+/settings/?$ {
    try_files /businesses/_/settings.html =404;
}
```

Don't touch `api.ranjbar.dev.conf`. This ships on the next deploy (CI syncs
`deploy/nginx/` to `/etc/nginx/conf.d/` and reloads) — no other action
needed.

---

## 2. Backend: admin avatar + self-service profile edit

### 2.1 Migration `api/db/migration/000005_admin_avatar.up.sql`

```sql
ALTER TABLE admins ADD COLUMN avatar BYTEA;
ALTER TABLE admins ADD COLUMN avatar_type TEXT NOT NULL DEFAULT '';
```

`000005_admin_avatar.down.sql`:

```sql
ALTER TABLE admins DROP COLUMN avatar_type;
ALTER TABLE admins DROP COLUMN avatar;
```

`// ponytail: avatar stored as bytea on the admins row — there are a handful
of super-admins, not thousands of users, so no object storage / volume
mount / nginx location is worth it. Revisit if the admins table ever grows
large or avatars get bigger than a small profile photo.`

### 2.2 Queries — `api/query/admin_auth.sql`, append:

```sql
-- name: SetAdminAvatar :exec
UPDATE admins SET avatar = $2, avatar_type = $3 WHERE id = $1;
-- name: ClearAdminAvatar :exec
UPDATE admins SET avatar = NULL, avatar_type = '' WHERE id = $1;
-- name: GetAdminAvatar :one
SELECT avatar, avatar_type FROM admins WHERE id = $1;
-- name: UpdateAdminProfile :one
UPDATE admins SET first_name = $2, last_name = $3, email = $4, phone_number = $5, is_male = $6
WHERE id = $1 RETURNING *;
```

Run `sqlc generate` in `api/` after adding these (existing `SELECT *`
queries like `GetAdminByID` automatically pick up the two new columns once
regenerated — check the generated `Admin` struct gets `Avatar []byte` /
`AvatarType string`).

### 2.3 Domain — `api/internal/domain/admin/admin.go`

Add `AvatarType string` to the `Admin` struct (mirrors what the DTO needs;
raw bytes never leave the repo/service layer except through the dedicated
binary endpoint). Wire it wherever the repo currently maps sqlc rows → domain
`Admin` (`api/internal/infrastructure/repo`, admin repo file).

### 2.4 Repo — admin repo file under `api/internal/infrastructure/repo`

Add methods: `SetAvatar(ctx, id int64, data []byte, contentType string) error`,
`ClearAvatar(ctx, id int64) error`,
`GetAvatar(ctx, id int64) (data []byte, contentType string, err error)`,
`UpdateProfile(ctx, id int64, firstName, lastName, email, phone string, isMale bool) (admin.Admin, error)`.
Follow the existing repo method style (error wrapping, sqlc calls) already
in that file.

### 2.5 Service — `api/internal/application/adminauth/service.go`

Add matching service methods that just delegate to the repo (same thin
pattern as the existing service methods there). Validate `contentType` is
one of `image/png`, `image/jpeg`, `image/webp` and `len(data) <= 1<<20` (1
MiB) in the service, returning a domain validation error the handler maps to
400 — follow whatever error type/mapping convention `adminauth.Service`
already uses elsewhere in that file (check how existing validation errors
are surfaced and returned as 400 in `admin_auth_handler.go` before adding a
new one).

### 2.6 Handler — `api/internal/interface/http/admin_auth_handler.go`

Add:
- `PATCH /admin/me` → `updateProfile(c)`: binds
  `{first_name,last_name,email,phone_number,is_male}` (same field names as
  `adminJSON`), calls service, returns `{"admin": adminJSON(updated)}`.
- `POST /admin/me/avatar` → `uploadAvatar(c)`: `c.FormFile("file")` (Gin
  multipart), read into `[]byte`, take `Content-Type` from the multipart
  file header, call service `SetAvatar`. Returns `{"avatar_url":
  "/admin/avatars/<id>?v=<unix millis>"}` (cache-busting query so the
  frontend can force a re-fetch after upload). On oversize/bad type return
  400 via `errJSON`.
- `DELETE /admin/me/avatar` → clears avatar, `204`.
- `GET /admin/avatars/:id` → `avatarPublic(c)`: **no auth** (avatars are not
  secret; `<img src>` cross-origin can't carry the Bearer header). Look up
  by `:id`, `404` if `avatar_type == ""`, else
  `c.Data(200, avatarType, bytes)`.

Update `adminJSON(a admin.Admin) gin.H` to add:
`"avatar_url": avatarURL(a)` where `avatarURL` returns `nil` if
`a.AvatarType == ""` else the string `"/admin/avatars/" + strconv.FormatInt(a.ID, 10)`
(no cache-buster needed here — the frontend appends its own `?v=` when it
knows the upload just changed).

### 2.7 Router — `api/internal/interface/http/router.go`

Inside the existing authenticated `p := r.Group("/admin"); p.Use(AdminAuth(tokens))`
block, add:
```go
p.PATCH("/me", a.updateProfile)
p.POST("/me/avatar", a.uploadAvatar)
p.DELETE("/me/avatar", a.uploadAvatar) // wire to the delete handler, not uploadAvatar — use the actual method name you gave it above
```
And **outside** any auth group (top-level `r`, next to `r.GET("/health", ...)`):
```go
r.GET("/admin/avatars/:id", a.avatarPublic)
```

### 2.8 Client body size

`multipart/form-data` avatar upload needs a body-size allowance. Check
`api/internal/config/config.go` / `router.go` for any existing
`MaxMultipartMemory` or body-limit setting; if Gin's default (32 MiB) is in
effect that's already enough headroom for a 1 MiB avatar — don't add new
config unless something in the codebase already caps it lower.

---

## 3. Frontend dependencies

In **both** `admin/` and `client/`:

```
npm install react-select react-multi-date-picker react-date-object
```

(`react-date-object` is `react-multi-date-picker`'s calendar/locale peer —
needed directly for the Persian calendar + locale imports below.)

---

## 4. Shared UI primitives (per app — `admin/components/` and
`client/components/`, same filenames/APIs in both, themed to that app's
tokens)

Create `admin/components/` and `client/components/` (new dirs, both apps
currently have none).

### 4.1 `Select.tsx`

Thin wrapper around `react-select`:

```tsx
"use client";
import RSelect, { Props as RSelectProps } from "react-select";

export type Option = { value: string; label: string };

export default function Select({ placeholder, ...props }: RSelectProps<Option, false> & { placeholder?: string }) {
  return (
    <RSelect
      {...props}
      isRtl
      placeholder={placeholder ?? "انتخاب کنید"}
      noOptionsMessage={() => "موردی یافت نشد"}
      classNamePrefix="rs"
      unstyled
      classNames={{
        control: (s) => `min-h-11 rounded-lg border px-1 text-sm ${s.isFocused ? "border-brand-accent ring-2 ring-brand-accent/30" : "border-brand-border"} bg-brand-bg`,
        menu: () => "mt-1 rounded-lg border border-brand-border bg-brand-surface text-sm shadow-lg overflow-hidden",
        option: (s) => `px-3 py-2 cursor-pointer ${s.isFocused ? "bg-brand-accent/10" : ""} ${s.isSelected ? "bg-brand-accent text-white" : ""}`,
        placeholder: () => "text-brand-muted",
        singleValue: () => "text-brand-text",
        input: () => "text-brand-text",
        indicatorSeparator: () => "hidden",
        dropdownIndicator: () => "text-brand-muted px-2",
      }}
    />
  );
}
```

Adjust class names/tokens if the exact `unstyled` + `classNames` API differs
from the installed `react-select` version — check `node_modules/react-select`'s
type defs after install and match its real API; the shape above is the
intended visual result (bordered field matching existing `<input>` styling,
surface-colored dropdown, accent-colored selected/hover option), not a
literal must-copy snippet.

### 4.2 `DatePicker.tsx`

Wraps `react-multi-date-picker` with the Persian calendar, single-value ISO
in/out:

```tsx
"use client";
import DP from "react-multi-date-picker";
import persian from "react-date-object/calendars/persian";
import persian_fa from "react-date-object/locales/persian_fa";

export function DatePicker({ value, onChange, placeholder }: { value: string | null; onChange: (iso: string | null) => void; placeholder?: string }) {
  return (
    <DP
      calendar={persian}
      locale={persian_fa}
      value={value ? new Date(value) : null}
      onChange={(d) => onChange(d ? (d as any).toDate().toISOString() : null)}
      inputClass="h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm focus-visible:ring-2 focus-visible:ring-brand-accent/30"
      placeholder={placeholder ?? "انتخاب تاریخ"}
      calendarPosition="bottom-right"
    />
  );
}

export function DateRangePicker({ value, onChange, placeholder }: { value: { from: string; to: string } | null; onChange: (v: { from: string; to: string } | null) => void; placeholder?: string }) {
  // range prop — see react-multi-date-picker range docs; onChange gives an
  // array of 2 DateObjects (or fewer while mid-selection). Convert both ends
  // to UTC day-start / day-end ISO strings (reuse the day-boundary logic
  // already in admin/lib/jalali.ts's jalaliDayRange, adapted to take a JS
  // Date instead of y/m/d — don't duplicate that logic).
}
```

Theme (both apps, in `globals.css`): override the library's own CSS
variables/classes so the calendar popover matches brand tokens instead of
its default blue theme — target `.rmdp-wrapper`, `.rmdp-calendar`,
`.rmdp-day.rmdp-selected span`, `.rmdp-day:hover span`, `.rmdp-header`,
`.rmdp-arrow-container` etc. Read the installed package's shipped CSS
(`node_modules/react-multi-date-picker/styles/`) to find the exact
selectors, then override background/border/text with `var(--color-brand-*)`
so it looks native to each app (dark surface in admin, white surface in
client), not a generic blue widget. Import the library's base CSS once in
each `globals.css` (`@import "react-multi-date-picker/styles/layout/prime.css";`
or whatever the installed version's minimal base layout file is called) then
layer the token overrides after it.

### 4.3 `Modal.tsx`

Fixes the native-`<dialog>` bug (not centered, no close button, backdrop
click does nothing — Tailwind v4's preflight zeroes the `<dialog>` UA
margin, which is what centers it by default) and replaces the ad-hoc
`<dialog>` in `admin/app/users/page.tsx`:

```tsx
"use client";
import { useEffect, useRef } from "react";

export default function Modal({ open, onClose, title, children }: { open: boolean; onClose: () => void; title: string; children: React.ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      onClick={(e) => { if (e.target === ref.current) onClose(); }}
      className="m-auto max-h-[85vh] w-full max-w-md overflow-y-auto rounded-2xl border border-brand-border bg-brand-surface p-6 text-brand-text backdrop:bg-black/50"
    >
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-bold">{title}</h2>
        <button type="button" aria-label="بستن" onClick={onClose} className="cursor-pointer rounded-lg p-1 text-brand-muted transition-colors hover:bg-brand-bg focus-visible:ring-2 focus-visible:ring-brand-accent/30">
          <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
        </button>
      </div>
      {children}
    </dialog>
  );
}
```

`m-auto` alone should already fix centering since it's specific enough to
beat preflight, but double-check the rendered result and add
`dialog[open] { margin: auto; }` to `globals.css` if any preflight rule
still wins — verify visually (or via the built HTML/CSS) that the dialog is
centered, not just that the class is present.

Rewrite `admin/app/users/page.tsx`'s inline `Modal` function to use this
shared component instead (state becomes a simple `open` boolean the parent
owns, passed to `<Modal open={open} onClose={() => setOpen(false)} title="کاربر جدید">`).

---

## 5. Admin layout — sidebar shell

### 5.1 Route group

Move these into a new `admin/app/(dash)/` route group (URLs unchanged —
groups don't affect paths):
`page.tsx` (dashboard), `users/`, `businesses/`, `settings/`. Add
`admin/app/(dash)/profile/page.tsx` (new, section 6). Leave `login/`,
`login/2fa/`, `forgot-password/`, `reset-password/` at the top level
(outside the group — no sidebar on auth screens).

### 5.2 `admin/app/(dash)/layout.tsx`

```tsx
"use client";
import Sidebar from "@/components/Sidebar";
export default function DashLayout({ children }: { children: React.ReactNode }) {
  return <Sidebar>{children}</Sidebar>;
}
```

### 5.3 `admin/components/Sidebar.tsx`

- `"use client"`, calls `useRequireAuth()` itself (redirects to `/login` if
  unauthenticated) so every wrapped page gets auth-guarding for free —
  **remove the now-redundant `useRequireAuth()` calls from the individual
  pages it wraps** (dashboard, users, users/[id], businesses,
  businesses/[id], settings/security) since the layout already guards them;
  those pages keep using the `admin` value if they need it, just get it
  from a shared context instead of calling the hook again — expose
  `admin`/`reload` via a React context (`AdminContext`, mirrors
  `client/app/businesses/[id]/Shell.tsx`'s `BusinessContext` pattern) so
  descendants can `useAdmin()` instead of re-fetching `/admin/me`.
- Sits fixed on the RTL leading edge (`right-0` — this is a right-side
  sidebar per the request), full viewport height, `bg-brand-surface`,
  border on its left (`border-l border-brand-border`).
- Two width states: expanded (`w-64`) and minimized/collapsed (`w-16`,
  icon-only, no text labels, tooltips via native `title=`). A toggle button
  in the sidebar header switches between them. Persist the collapsed
  boolean in `localStorage` (`admin_sidebar_collapsed`), read on mount
  (guard for SSR — static export still prerenders once, so wrap
  `localStorage` reads in `typeof window !== "undefined"`).
- Also collapsible to fully hidden on small screens (a hamburger toggle in
  a slim top bar that appears only when the sidebar is hidden/off-canvas)
  — reuse the same collapsed/expanded mechanism responsively; don't build a
  second separate mobile drawer component, just let `w-16`/`w-64`/`w-0`
  (translate off-screen) share one state machine driven by one toggle
  button, with a `max-sm:` breakpoint defaulting to hidden.
- Header block (top of sidebar): avatar (`<img src={${API_BASE}/admin/avatars/${admin.id}} onError={...fallback}>`
  if `admin.avatar_url`, else a `--color-brand-accent`-colored circle with
  the admin's first+last initials), admin's `first_name last_name`, and
  `phone_number` (dir="ltr" on the number, matches existing table cells).
  Hide the text block (keep only the avatar, centered) when collapsed.
- Nav links (icon + label, label hidden when collapsed):
  کاربران → `/users`, کسب‌وکارها → `/businesses`, پروفایل → `/profile`.
  Active route highlighted via `usePathname()` (same active-state pattern
  as `client/app/businesses/[id]/Shell.tsx`'s `nav`).
  Simple inline `<svg>` icons (stroke-based, matching the existing
  plus-icon style already used in `users/page.tsx` — don't add an icon
  library for three icons).
- Footer: تنظیمات امنیتی → `/settings/security`, and خروج (logout) button —
  reuse the exact logout logic already in `admin/app/page.tsx`
  (`apiFetch("/admin/auth/logout", {method:"POST"})` then `clearSession()`
  then redirect to `/login`).
- Main content wrapper: `<main>` with `mr-64`/`mr-16`/`mr-0` (RTL — margin
  on the right matches sidebar width) transitioning with the sidebar state,
  `min-h-screen`.

### 5.4 Dashboard — `admin/app/(dash)/page.tsx`

Strip it down to just the page shell with **no cards/stats/links** (sidebar
now owns navigation) — a bare heading is fine (`خوش آمدید، {admin.first_name}`
via `useAdmin()`), everything else empty per the request ("for now ... put
nothing, I will add statistic analysis data there"). Remove the old
`<header>`/logout button/link-buttons entirely — sidebar replaces all of
that.

---

## 6. Profile page — `admin/app/(dash)/profile/page.tsx`

New page, uses `useAdmin()` context from the sidebar. Shows/edits own
profile:
- Avatar: current image (or initials placeholder), a file `<input
  type="file" accept="image/png,image/jpeg,image/webp">` that immediately
  uploads on change (`POST /admin/me/avatar` as `multipart/form-data`,
  `FormData` with the file under key `"file"`), toast on success/error,
  refresh the avatar (`?v=Date.now()` cache-bust) and the shared
  `AdminContext` so the sidebar picks up the new avatar without a reload. A
  "حذف تصویر" button when an avatar exists → `DELETE /admin/me/avatar`.
- Editable fields (reuse the existing `FormField` component from
  `admin/app/users/page.tsx` — move it to `admin/components/FormField.tsx`
  since it's now shared across users, users/[id], and profile — check
  `admin/app/users/[id]/Detail.tsx` for whether it already has its own
  copy/variant and consolidate to one shared component rather than three
  copies): نام، نام خانوادگی، ایمیل، موبایل. جنسیت (male/female) via the new
  `Select` component. Submit → `PATCH /admin/me`, toast, update context.
- A link to «تنظیمات امنیتی» for password/2FA (that page already exists,
  don't duplicate its functionality here).
- `lib/adminProfile.ts` (or add to existing `lib/`) with `updateProfile`,
  `uploadAvatar`, `deleteAvatar` functions following the same
  fetch-wrapper style as `admin/lib/users.ts`.

---

## 7. Swap native controls → `Select` / `DatePicker`

Replace every native `<select>` and the ad-hoc 3-`<select>` date filter in
`admin/`:
- `admin/app/users/page.tsx`: `account_type` select in the create modal,
  `status` filter select, and the entire `DateFilter` component (3 selects)
  → single `DateRangePicker` from section 4.2, emitting the same
  `{from,to}` shape `Filters["range"]` already expects (keep
  `admin/lib/jalali.ts` for table-cell display formatting via
  `isoToJalaliLabel` — don't remove it).
- `admin/app/users/[id]/Detail.tsx` line ~92: `account_type` select.
- `admin/app/businesses/[id]/Detail.tsx`: per-row member role select.
- `admin/app/businesses/page.tsx`: if the owner-picker uses a select
  anywhere (check — it may be radio buttons per the CLAUDE.md notes; if so
  leave radios alone, that's not a `<select>`).

Each replacement keeps the exact same value/onChange contract the
surrounding component already has — this is a UI-layer swap, not a data
model change.

---

## 8. Placeholders

Add a meaningful Persian `placeholder` to **every** text/tel/email/password
`<input>` and every `Select`/`DatePicker` across both apps (the two admin
auth pages, forgot/reset-password, 2FA code input, the users create modal
+ filter row, users/[id] edit fields, businesses create/rename/add-member
fields, businesses/[id] fields, settings/security fields, client login/2FA/
forgot/reset, client business create/rename/invite forms). Example:
`placeholder="۰۹۱۲۱۲۳۴۵۶۷"` for a phone field, `placeholder="مثال:
احمد"` for a first-name field, `placeholder="example@mail.com"` for email.
Pick sensible text per field — don't leave any input without one.

---

## 9. Client (`client/`)

- Add the same `client/components/Select.tsx` and
  `client/components/DatePicker.tsx` (section 4.1/4.2), themed to the
  **client** tokens (light surface, blue accent) instead of admin's.
- Swap every native `<select>` in `client/app/**` to the new `Select`:
  `client/app/businesses/[id]/Shell.tsx`'s business switcher, and the role
  selects in `client/app/businesses/[id]/members/page.tsx` (invite role +
  per-member role change).
- Client has no existing date inputs — just make sure `DatePicker`/
  `DateRangePicker` are ready (built + themed) for when a client screen
  needs one; no page changes required beyond the `Select` swap above.
- Same placeholder sweep as admin (section 8) across client's forms.
- Don't touch `client/app/businesses/[id]/Shell.tsx`'s overall layout —
  only the `<select>` element mentioned above. The sidebar/profile/avatar
  work in this plan is admin-only; client's shell already has its own
  header nav and is out of scope.

---

## 10. Acceptance criteria

- `cd admin && npm run build` — 0 errors, static export succeeds.
- `cd client && npm run build` — 0 errors, static export succeeds.
- `cd api && go build ./...` — succeeds. `go vet ./...` clean.
- `~/go/bin/migrate -path api/db/migration -database <local dsn> up` applies
  `000005` cleanly (test against the local docker compose postgres).
- Manually verify (or describe how you verified, since this may run
  headless): create-user modal opens centered with a working X button and
  backdrop-click-to-close; admin sidebar toggles/minimizes and persists
  across reload; dashboard body is empty; profile page uploads an avatar
  and the sidebar reflects it without a manual refresh; every swapped
  `<select>`/date filter still filters/submits the same as before.

---

## 11. Docs — append to learnings logs (mandatory, don't skip)

Append one dated bullet (`2026-08-30 — ...`) to the "Learnings log" section
of **root** `CLAUDE.md` recording: the sidebar layout pattern
(`(dash)` route group + `Sidebar.tsx` + `AdminContext`), that `Modal.tsx` is
now the standard modal (native `<dialog>` + `m-auto` fix, backdrop-click +
X close) to reuse instead of ad-hoc dialogs, that `react-select` and
`react-multi-date-picker` are now the standard select/date-picker libs for
both apps (themed per-app in `components/Select.tsx` /
`components/DatePicker.tsx` against the `--color-brand-*` tokens — new
screens should use these, not native `<select>`/date inputs), the avatar
storage decision (bytea column + public unauthenticated
`GET /admin/avatars/:id`, 1 MiB / png-jpeg-webp only), and the nginx
`[id]`-route rewrite fix (mirror `admin/serve.json` / `client/serve.json`
into the matching `deploy/nginx/*.conf` whenever a new dynamic route is
added, or prod 404s/misroutes on deep-link).

Add a one-line bullet to `admin/CLAUDE.md`'s Learnings section pointing at
the new `admin/components/` primitives and the `(dash)` route group, mirroring
the style of its existing entries.
