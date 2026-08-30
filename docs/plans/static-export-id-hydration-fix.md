# Fix: hard-loading a dynamic static-export route lands back on the list

## Confirmed root cause

`https://admin.ranjbar.dev/users/1` (and `/businesses/1`, and client
`/businesses/1/{dashboard,members,settings}`) redirect back to the list page
on a hard load / deep link, even though the nginx rewrite (already fixed
separately) correctly serves the prebuilt `_.html` shell.

Reproduced locally: built `admin`, served `out/` with `serve -c serve.json`
(same `try_files`-style rewrite nginx now does), and inspected the HTML
response for `GET /users/1`. The embedded Next.js flight payload contains:

```
"c":["","users","_"]
```

— the canonical route state Next bakes into the static HTML at build time,
using the `generateStaticParams` placeholder (`"_"`), not the real requested
URL. On hydration, `useParams()` in the `"use client"` detail component
resolves the `[id]` segment from that baked state, so it reads `"_"`, not
`"1"`. `Number("_")` is `NaN`, and the page's own invalid-id guard
(`if (!Number.isInteger(id) || id < 1) router.replace(...)`) fires and sends
the user back to the list. This only breaks the *first* hard load — a
client-side `router.push`/`<Link>` navigation from the list already works
today because Next's client router re-derives params from the real URL for
soft navigations; only the initial hydration of a directly-loaded page uses
the baked placeholder.

## Fix

In each of the three places that derive a dynamic-route id from
`useParams()` at the top of a `"use client"` page/component, fall back to
parsing the real browser URL when the param is the static-export placeholder
(`"_"`). This is synchronous (no extra `useEffect`/state) and safe: in every
affected file the id is only consumed inside a `useEffect`/event handler,
never rendered directly before the component's own loading gate, so there is
no hydration-mismatch risk.

### 1. `admin/app/(dash)/users/[id]/Detail.tsx`

Replace:
```ts
const id = Number(useParams<{ id: string }>().id);
```
with:
```ts
const paramId = useParams<{ id: string }>().id;
// ponytail: output:"export" bakes the generateStaticParams placeholder
// ("_") into the hydration payload instead of the real URL segment, so
// useParams() reports "_" on a hard load / deep link. Fall back to the
// real browser path in that one case; client-side navigation already
// resolves the real id via useParams() and is untouched.
const id = Number(
  paramId && paramId !== "_"
    ? paramId
    : (typeof window !== "undefined" ? window.location.pathname.match(/^\/users\/([^/]+)/)?.[1] : undefined)
);
```

### 2. `admin/app/(dash)/businesses/[id]/Detail.tsx`

Same pattern, replacing `const id=Number(useParams<{id:string}>().id)` — use
the regex `^/businesses/([^/]+)` (this file is minified/one-line; keep it
consistent with the surrounding style, but the logic must match #1 exactly
except for the regex).

### 3. `client/app/businesses/[id]/Shell.tsx`

Replace `const{id}=useParams<{id:string}>(),bid=Number(id)` with the same
fallback pattern (regex `^/businesses/([^/]+)`), producing `bid`. This is
the one place in the client app that actually parses the URL — fixing it
here fixes the family of nested routes below it once you also do #4.

### 4. `client/app/businesses/[id]/members/page.tsx` and
   `client/app/businesses/[id]/settings/page.tsx`

Both independently re-derive `bid` via their own `useParams()` call instead
of trusting `Shell.tsx`, which by the time these render has already
successfully loaded the business (so its id is known-good). Replace:
```ts
const bid=Number(useParams<{id:string}>().id)
```
with reading it off the existing business context instead — both files
already call `useBusiness()`:
```ts
const {business} = useBusiness(); // already imported from "../Shell"
const bid = business.id;
```
(`members/page.tsx` also uses `role`; `settings/page.tsx` already destructures
`business` — just add `.id` access, don't reintroduce a `useParams` call.)
Remove the now-unused `useParams` import from both files if nothing else in
them needs it (`Members` uses `useRouter`, keep that).

## Verification (do this yourself, don't just build)

1. `cd admin && npm run build`, then serve the static output with the same
   rewrite prod nginx uses: `npx --yes serve out -l 3011 -c ../serve.json`.
   `curl -s http://localhost:3011/users/1 | grep -o '"c":\["","users","[^"]*"\]'`
   will still show `"_"` (that's the static HTML, expected and harmless) —
   the real test is behavioral: local Postgres + API are already running
   (`docker compose ps` should show `hesab-api-1` and `hesab-postgres-1`
   up on port 8080; admin's default `NEXT_PUBLIC_API_URL` already points at
   `http://localhost:8080`). Use a browser tool to open
   `http://localhost:3011/login`, sign in with seed admin `9370843199` /
   `Amir@Pass1999`, then **directly navigate** (type the URL, don't click
   through — a real new navigation, not a client-side `<Link>` transition)
   to `http://localhost:3011/users/1`. Confirm it renders the user detail
   page and the URL stays `/users/1` — it must NOT bounce back to `/users`.
   Repeat for a `/businesses/<id>` id that exists.
2. `cd client && npm run build`, serve the same way
   (`npx --yes serve out -l 3012 -c ../serve.json`), sign in with seed
   client `9120000000` / `Client@Pass1999`, directly navigate to
   `http://localhost:3012/businesses/<id>/members` for a business the seed
   client belongs to. Confirm it renders (doesn't bounce to
   `/select-business`).
3. `go build ./...` is untouched by this change (frontend-only) — just
   confirm nothing under `api/` was touched.

Report exactly what you observed for both direct-navigation tests (URL
before/after, what rendered) — don't just report "build passed".
