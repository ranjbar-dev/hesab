# برنامه حساب‌داری — Accounting SaaS

Persian (RTL) multi-tenant accounting software.

## Product

- **Admin panel** (`./admin`) — super-admin manages companies and their online
  subscriptions (plans, billing, activation/suspension).
- **Client dashboard** (`./client`) — companies and individuals manage their
  own accounting data.
- Persian UI, RTL layout, Jalali (Shamsi) dates.

## Repo layout

| Path            | Contains                                                    |
|-----------------|------------------------------------------------------------|
| `./api`         | Go backend — Gin + sqlc + JWT, DDD layered architecture     |
| `./admin`       | Next.js super-admin panel                                   |
| `./client`      | Next.js dashboard for companies / individuals               |
| `./db/postgres` | Postgres image config, init SQL, tuning                     |
| `./` (root)     | `docker-compose.yml`, `.env`, shared project-level files    |

## Stack

- **Backend:** Go, Gin (HTTP), sqlc (Postgres access, no ORM), JWT (client
  auth), DDD design pattern.
- **Frontend:** Next.js `16.3.3`, Tailwind CSS v4, App Router.
- **Database:** PostgreSQL.
- **Infra:** Docker + Docker Compose.

## Workflow — MANDATORY

Claude Code is the **orchestrator**. Codex CLI is the **coder**. Claude does
not write feature code directly.

For every task, in order:

1. **Brainstorm first.** Before running Codex, ask the user clarifying
   questions (brainstorm-skill style) until the task and every requirement is
   fully understood. Do not assume.
2. **Plan.** Write a detailed implementation plan for Codex: files to touch,
   endpoints, DB schema, layer placement, acceptance criteria.
3. **Delegate.** Spawn Codex CLI with that plan to write the code.
4. **Review.** Claude reviews Codex output before the task is considered done.

### Git worktrees — MANDATORY for feature / code work

Multiple agents may work this repo at once. Never edit features directly on
`main`.

1. If `./` is not a git repo yet, `git init` and make an initial commit on
   `main` first.
2. Per task, create an isolated worktree off `main`:
   `git worktree add ../hesab-<task-slug> -b feat/<task-slug>`
3. All Codex delegation, edits, and review for that task happen inside that
   worktree — keeps concurrent agents from colliding.
4. When the task is done and reviewed: merge the branch into `main`
   (`git -C . merge --no-ff feat/<task-slug>`), then
   `git worktree remove ../hesab-<task-slug>` and delete the branch.
5. One worktree per task. Don't reuse a worktree across unrelated tasks.

## Always-on skills / tools

- **ponytail** — laziest solution that works. Every task.
- **RTK** — token-optimized CLI proxy. Every shell operation.
- **caveman** — terse output. Every task.
- **ui-ux-pro-max** — every frontend task in `./admin` and `./client`.

## Conventions

**Ports** (overridable via root `.env`): `api` 8080, `postgres` 5432,
`admin` 3010, `client` 3020.

**Frontend design tokens** — keep new screens consistent with these. Defined
as `--color-brand-*` in each app's `app/globals.css` `@theme` block; both apps
use the Vazirmatn webfont and `dir="rtl"`.

| | admin | client |
|---|---|---|
| Mode | dark | light |
| Background | `#0F172A` | `#F8FAFC` |
| Surface | `#1E293B` | `#FFFFFF` |
| Border | `#334155` | `#E2E8F0` |
| Text | `#F8FAFC` | `#1E293B` |
| Muted | `#94A3B8` | `#64748B` |
| Accent | amber `#F59E0B` | blue `#2563EB` |

## Learnings log — READ before planning, APPEND after fixes

If the user fixes or changes something Codex or Claude produced, and it must
not happen again, append a dated bullet here (or to the relevant folder's
`CLAUDE.md` if it is folder-specific). Check this list before planning any
related work.

<!-- LEARNINGS -->
- **2026-08-30 — Client authentication.** Client auth deliberately parallels `adminauth` while staying separate: it uses `users`, `/client/auth/*` routes, and the `client_refresh_token` HttpOnly cookie scoped to `/client/auth`; the seed user is `9120000000` / `Client@Pass1999`.
- **2026-08-30 — Go module version.** `go mod tidy` pulls a test-dep chain
  (gin → testify → rogpeppe/go-internal) that needs Go ≥ 1.25, so `api/go.mod`
  is `go 1.25` and `api/Dockerfile` uses `golang:1.25-alpine`. Do not
  downgrade either or the Docker build breaks.
- **2026-08-30 — Frontends are static-export SPAs.** `admin` and `client` use
  `output: "export"` in `next.config.ts` (no Node server, no SSR). They are
  NOT in `docker-compose.yml`. Run locally: `npm install && npm run dev`
  (`admin` → 3010, `client` → 3020). Design tokens live as `--color-brand-*`
  in each `app/globals.css` `@theme` block; Persian text uses the Vazirmatn
  webfont.
- **2026-08-30 — ui-ux-pro-max script on Windows.** Its `search.py` crashes on
  cp1252 consoles. Prefix commands with `export PYTHONIOENCODING=utf-8`.
- **2026-08-30 — Windows console mangles UTF-8 in test payloads.** Persian sent
  through `curl -d '{...}'` or printed from a `python -c`/heredoc lands as `???`
  (cp1252), independent of the API — the API + Postgres are UTF-8 clean.
  Verifying a Persian round-trip means checking it in `psql` inside the DB
  container, or asserting a filter's match count, not reading the shell echo.
  `PYTHONIOENCODING=utf-8` fixes Python stdout but not `curl`'s request body.
- **2026-08-30 — Next 16 workspace root.** A `package-lock.json` in `C:\Users\root`
  makes Turbopack guess the wrong root. Both `next.config.ts` files pin
  `turbopack: { root: import.meta.dirname }`. Keep it.
- **2026-08-30 — Next 16 rewrites files on `next dev`.** It edits `tsconfig.json`
  (`jsx: react-jsx`, extra `include`), `next-env.d.ts`, and appends a
  `<!-- BEGIN:nextjs-agent-rules -->` block to `admin/AGENTS.md` +
  `client/AGENTS.md`. Expected — commit these, don't fight them. IDE TS
  diagnostics may lag after this; `next build` (0 errors) is the source of truth.
- **2026-08-30 — Delegating to Codex.** Invocation used:
  `codex exec --skip-git-repo-check --dangerously-bypass-approvals-and-sandbox
  -C <worktree-abs-path> -o <outfile> "<prompt>"` where the prompt tells it to
  read a plan file under `docs/plans/`. Runs unattended and needs network for
  `go get` / `npm install`. The repo now has commits, so
  `--skip-git-repo-check` is no longer required (still harmless inside a
  worktree). `-C` MUST be the real worktree path: from
  `C:\Users\root\Desktop\hesab`, `git worktree add ../hesab-<slug>` creates
  `C:\Users\root\Desktop\hesab-<slug>` (a sibling of the repo, NOT under
  `C:\Users\root`). Write the plan file *inside* that worktree — `Write`
  silently creates missing parent dirs, so a wrong path just makes a stray dir
  containing only the plan and Codex reports the repo missing. Codex takes the
  cheap way around a hard constraint unless the plan blocks it: told to build a
  `/users/$id` screen in a `output: "export"` SPA it shipped `/users/detail?id=`
  (a query param) rather than a `[id]` route. Put the exact route shape AND the
  workaround recipe in the plan (see the admin static-export dynamic-route
  learning), then check the delegated output actually followed it.
- **2026-08-30 — Check sibling worktrees before planning a shared-table feature.**
  Multiple agents run this repo at once (`git worktree list`). Before planning
  work that touches a shared table/domain, `cd` into each sibling worktree and
  read its `git status` + any new `api/db/migration/*` / `api/query/*` — an
  in-progress branch may already be creating the table you need. `feat/users-admin`
  and `feat/client-auth` both needed `users`; had client-auth not merged first,
  the parallel plan would have collided on migration `000002` and the `user`
  domain package.
- **2026-08-30 — RTK wraps some commands.** `docker logs <c>` comes back as a
  summarised "Log Summary" — use `rtk proxy docker logs <c>` for raw lines.
  `find` with `-not` / `-exec` fails under RTK ("does not support compound
  predicates"); use plain `find` predicates or the Glob/Grep tools.
  `go test ./...` returns a `"Go test: N passed in M packages"` summary, not
  per-test lines — use `rtk proxy go test ... -v` to see individual tests or a
  failure's detail.
- **2026-08-30 — User-facing messages use toasts.** `admin` and `client` both
  depend on `sonner` (`^2.0.8`). Each `app/layout.tsx` renders one `<Toaster
  dir="rtl" richColors position="top-center">` (admin `theme="dark"`, client
  `theme="light"`, `toastOptions.style.fontFamily:"inherit"` for Vazirmatn).
  For any error / warning / info / success feedback (login, forgot/reset
  password, 2FA, etc.) call `toast.error|warning|info|success("پیام فارسی")` —
  do NOT add inline `<p className="text-red-300">` message state to forms.
- **2026-08-30 — Kill local `go run` servers before `git worktree remove`
  (Windows).** A `go run ./cmd/server` started inside a worktree for
  smoke-testing leaves a `server.exe` holding `api/`; `git worktree remove`
  then fails the physical delete ("Device or resource busy" /
  "Permission denied") even though git still unregisters the worktree.
  `pkill -f 'cmd/server'` misses it — use `taskkill //F //IM server.exe`, then
  `rm -rf` the dir and `git worktree prune`.
- **2026-08-30 — Smoke-test worktree API without rebuilding the container.**
  `hesab-api-1` holds `:8080` with a possibly-stale image. To exercise
  worktree backend code, run `PORT=8081 COOKIE_SECURE=false go run ./cmd/server`
  from the worktree's `api/` against the shared local Postgres (`:5432`, from
  `docker compose`). Migrations/seed hit that same DB, so `migrate ... up` may
  report "no change" if a sibling worktree already applied them.
- **2026-08-30 — CRLF churn: modified in `git status`, empty `git diff`.**
  `core.autocrlf` on this box makes freshly written / `sqlc generate`d files
  show as modified with no content diff (LF→CRLF warnings on `git add`).
  `git checkout -- <path>` clears it; it is not a real change, don't commit it.
- **2026-08-30 — Businesses & memberships.** `feat/businesses` added the
  `businesses` (1:N `owner_user_id`), `business_members` (roles `owner` /
  `admin` / `accountant` / `viewer`; owner also gets a member row) and
  `business_invites` (`pending`/`accepted`/`rejected`/`cancelled`, partial
  unique on pending) tables — migration `000004`. Client invites are by phone
  and create a pending invite the invitee accepts on next login; admin adds
  members immediately. `owner`+`admin` manage members; only `owner` (or
  super-admin) soft-deletes a business. Client app is now business-scoped:
  routes live under `/businesses/[id]/{dashboard,members,settings}`,
  `/select-business` picks one (0 → empty state, exactly 1 → auto-enter, 2+ →
  picker + invites inbox), the header `<select>` switches business by changing
  the URL id only (nothing persisted). Codex delegation of a big multi-layer
  task (API + both SPAs) came back complete and building first try when the
  plan spelled out JSON shapes, error→HTTP mapping and the static-export
  route pattern per page.
- **2026-08-30 — Deploy: GitHub Actions to the Linux server.**
  `.github/workflows/deploy.yml` fires on push to `main`, path-filtered
  (`dorny/paths-filter`) into 4 independent jobs. `api` (also `db/**`,
  `docker-compose.yml`): rsync the whole repo to `/opt/hesab/api` (excludes
  `.git .github node_modules admin client .env`), `docker compose up -d
  --build`. `admin` / `client`: `npm ci && npm run build` (static export),
  rsync `out/` to `/opt/hesab/{admin,client}`. `nginx` (`deploy/nginx/**`):
  see the conf.d bullet below. SSH is password auth via `sshpass -e`
  (`StrictHostKeyChecking=no`). Prod domains: client `ranjbar.dev`, admin
  `admin.ranjbar.dev`, api `api.ranjbar.dev` (nginx configs in
  `deploy/nginx/*.conf`, HTTP-only, api one is a reverse proxy to
  `127.0.0.1:8080`). 7 repo secrets: `SSH_HOST`, `SSH_USER`, `SSH_PASSWORD`,
  `SSH_PORT`, `ENV_API` (full prod `.env`, lands at `/opt/hesab/api/.env`),
  `ENV_ADMIN`, `ENV_CLIENT` (each just `NEXT_PUBLIC_API_URL=https://api.ranjbar.dev`,
  written to `<app>/.env.production` before build). `ENV_API` must set
  `CORS_ORIGINS=https://ranjbar.dev,https://admin.ranjbar.dev`,
  `COOKIE_DOMAIN=.ranjbar.dev`, `COOKIE_SECURE=true`.
- **2026-08-30 — Deploy: migration path on the server is doubled.**
  The whole repo rsyncs *under* `/opt/hesab/api/`, so migrations end up at
  `/opt/hesab/api/api/db/migration` (double `api`). The `api` job runs them
  after `compose up` with the `migrate/migrate` Docker image,
  `--network host`, DB URL built from the deployed `.env` by reading keys
  individually (`grep -E '^POSTGRES_USER=' .env | cut -d= -f2-`) — do NOT
  `source`/`.` the `.env`: `TOTP_ISSUER=Hesab Admin` has an unquoted space and
  sourcing it runs `Admin` as a command. Seeding is not in CI (run
  `cmd/seed` once by hand).
- **2026-08-30 — Deploy: `/etc/nginx/conf.d/` is repo-managed and `--delete`d.**
  The `nginx` job does `rsync -az --delete --rsync-path="sudo rsync"
  deploy/nginx/ → /etc/nginx/conf.d/` then `sudo nginx -t && sudo systemctl
  reload nginx`. Any `.conf` there NOT in `deploy/nginx/` is deleted (RHEL's
  `default.conf`, certbot output). `SSH_USER` needs NOPASSWD sudo for `rsync`,
  `nginx`, `systemctl reload nginx`. TLS is not wired — `certbot --nginx`
  would write into these files and be wiped next deploy; use `--webroot` + a
  separate non-managed include, or bake cert paths into the committed
  `.conf` files.

## Current state (2026-08-30)

`docker compose up -d --build` brings up `postgres` + `api`; `GET
http://localhost:8080/health` → `200 {"database":"up","status":"ok"}`.
Migrations are CLI-only (`~/go/bin/migrate ... up`); the container runs
neither migrate nor the seeders.

- `api` — Gin + pgxpool, DDD layers under `internal/`. Wired: `/health`,
  admin auth + 2FA (`/admin/auth/*`, `/admin/2fa/*`), client auth + 2FA
  (`/client/auth/*`, `/client/2fa/*`), admin user management (`/admin/users*`),
  businesses + memberships + invites (`/admin/businesses*`,
  `/admin/users/:id/businesses`, `/client/businesses*`, `/client/invites*`).
  Migrations `000001` admin_auth, `000002` client_auth, `000003` users_admin,
  `000004` businesses. Fake fixed-code SMS (`123456`); sms.ir still a TODO.
- `admin` — auth pages + `/settings/security` + `/users` + `/users/[id]`
  (now also lists the user's owned/joined businesses) + `/businesses`
  (list/search/paginate/create) + `/businesses/[id]` (rename, members CRUD,
  soft-delete), all wired cross-origin. `next build` static export clean.
- `client` — auth pages + business-scoped app: `/select-business` picker,
  `/businesses/new`, `/businesses/[id]/{dashboard,members,settings}` under a
  shared `Shell` with a business switcher. Dashboard body is still a stub.
- Seed users: admin `9370843199` / `Amir@Pass1999`; client `9120000000` /
  `Client@Pass1999`. No CI, no billing/subscription domain yet.
