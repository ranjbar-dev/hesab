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
- **2026-08-30 — Next 16 workspace root.** A `package-lock.json` in `C:\Users\root`
  makes Turbopack guess the wrong root. Both `next.config.ts` files pin
  `turbopack: { root: import.meta.dirname }`. Keep it.
- **2026-08-30 — Next 16 rewrites files on `next dev`.** It edits `tsconfig.json`
  (`jsx: react-jsx`, extra `include`), `next-env.d.ts`, and appends a
  `<!-- BEGIN:nextjs-agent-rules -->` block to `admin/AGENTS.md` +
  `client/AGENTS.md`. Expected — commit these, don't fight them. IDE TS
  diagnostics may lag after this; `next build` (0 errors) is the source of truth.

## Current state (2026-08-30)

Base scaffold only. Verified: `docker compose up -d --build` brings up
`postgres` + `api`; `GET http://localhost:8080/health` → `200
{"database":"up","status":"ok"}`.

- `api` — Gin + pgxpool, DDD layers under `internal/`, only `/health` wired.
  `sqlc.yaml` present, no queries/migrations yet.
- `admin` / `client` — one dummy login page each (not wired to the API),
  built with ui-ux-pro-max direction. `npm install` done, `npm run dev` +
  `next build` (static export to `out/`) both verified clean (0 errors).
  admin → :3010, client → :3020, pages serve `lang="fa" dir="rtl"`.
- No auth, no domain tables, no CI. Next task starts real features.
