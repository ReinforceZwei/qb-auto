# qb-auto Web UI — Frontend Design Plan

Source of truth: GitHub issue #13 (this replaces the outdated `web-ui.md`).

## 1. Goal

A web UI for qb-auto, embedded in the Go binary via `go:embed`, providing:

1. **Monitor jobs** — live view of active jobs with realtime updates
2. **Restart failed job** — manually fill required info and skip straight to rsync
3. **View job history** — browse all past jobs with filtering
4. **API playground** — ad-hoc requests against the backend API
5. **Login** — PocketBase superuser account

## 2. Tech stack (from issue)

| Concern        | Choice                          |
| -------------- | ------------------------------- |
| Build tool     | Vite                            |
| UI framework   | React 19 + TypeScript           |
| Routing        | Tanstack Router                 |
| Component lib  | Mantine (v8)                    |
| Backend client | PocketBase JS SDK (v0.22+)      |
| State          | Local + PocketBase realtime     |

The existing `ui/` folder is still the default Vite template — it will be re-scaffolded with the stack above (deps: `@tanstack/react-router`, `@mantine/core`, `@mantine/hooks`, `@mantine/notifications`, `pocketbase`, plus `@tabler/icons-react` for icons).

## 3. Project structure (`ui/`)

```
ui/
  index.html
  package.json
  tsconfig.json / tsconfig.app.json / tsconfig.node.json
  vite.config.ts              # base: '/', build outDir 'dist'
  src/
    main.tsx                  # MantineProvider + RouterProvider
    router.tsx                # Tanstack Router instance + route tree
    theme.ts                  # Mantine theme (dark/light, brand color)
    lib/
      pocketbase.ts           # singleton PB client + auth helpers
      types.ts                # Job type, JobStatus enum, status metadata
      api.ts                  # typed wrappers around custom /api/* routes
      realtime.ts             # jobs subscription hook (useJobsRealtime)
    components/
      layout/AppLayout.tsx    # AppShell: header + navbar + <Outlet/>
      JobStatusBadge.tsx      # colored status pill
      JobTable.tsx            # reusable jobs table (used by monitor + history)
      RetryJobModal.tsx       # manual retry form (skip to rsync)
      Playground.tsx          # API playground
    routes/
      __root.tsx              # root layout w/ auth guard
      login.tsx               # /login
      index.tsx               # /            — monitor
      history.tsx             # /history     — job history
      playground.tsx          # /playground  — API playground
      not-found.tsx           # 404
```

## 4. Backend data model (jobs)

Fields the UI consumes (from migrations):

| Field           | Type   | Notes                              |
| --------------- | ------ | ---------------------------------- |
| `id`            | string | PB record id                       |
| `status`        | string | see state machine below            |
| `torrent_hash`  | string |                                    |
| `category`      | string | `anime` or other/empty             |
| `anime_title`   | string | filled by title worker             |
| `anime_list_id` | string |                                    |
| `tmdb_id`       | number |                                    |
| `tmdb_season`   | number |                                    |
| `error`         | string | failure message                    |
| `completed`     | date   | set when done                      |
| `created`       | date   |                                    |
| `updated`       | date   |                                    |

### Status state machine

```
                    ┌─────────── error (terminal, retryable)
                    │
pending ──▶ processing_title ──▶ pending_rsync ──▶ processing_rsync ──▶ pending_notify ──▶ processing_notify ──▶ done
(non-anime starts at pending_notify)
```

### Status → UI presentation

| Status               | Tone / color | Meaning                       |
| -------------------- | ------------ | ----------------------------- |
| `pending`            | blue         | queued for title resolution   |
| `processing_title`   | indigo       | resolving title (LLM/TMDb)    |
| `pending_rsync`      | cyan         | queued for transfer           |
| `processing_rsync`   | teal         | transferring via rsync        |
| `pending_notify`     | violet       | queued for webhook            |
| `processing_notify`  | grape        | sending webhook               |
| `done`               | green        | completed                     |
| `error`              | red          | failed — retry action shown   |

A single `STATUS_META` map in `types.ts` drives `JobStatusBadge`, filters, and legends.

## 5. Layout (Mantine AppShell)

- **Header** (per issue): brand text `qb-auto` — no logo. Right side: version/status, logout button (superuser auth state).
- **Navbar**: collapsible (responsive) navigation with icon + label items:
  - Monitor (`/`)
  - Job history (`/history`)
  - API playground (`/playground`)
- **Main**: renders the active route via `<Outlet/>`, padded content area.
- Footer (optional): connection status + realtime indicator.

## 6. Pages

### 6.1 Login — `/login`
- Centered card with Mantine `PasswordInput` + `TextInput`.
- Authenticates against superuser collection: `pb.collection('_superusers').authWithPassword(email, password)`.
- On success → redirect to `/`; on failure show `Notifications` error.
- Root layout guards all other routes: no `pb.authStore.isValid` → redirect to `/login`.
- A "sign out" action clears `pb.authStore.clear()` and returns to login.

### 6.2 Monitor (jobs) — `/`
Realtime dashboard for **active / non-terminal** jobs.

- **Summary cards** (Mantine `SimpleGrid`): Active jobs, Processing now, Failed (error), Done today.
- **Live jobs table** (`JobTable`): columns — Status badge, Anime title (or torrent hash fallback), Category, Torrent hash, Created, Updated, Error (truncated w/ tooltip), Actions.
- **Realtime**: `pb.collection('jobs').subscribe('*', handler)` in a `useJobsRealtime` hook:
  - `create` / `update` events upsert rows live; `delete` removes them.
  - Default filter: `status != 'done' && status != 'error'` (terminal jobs move to history view).
  - A toggle lets the user "include done/error" if desired; reconnect handled by the SDK.
- **Actions column**: for `error` rows show **Restart** button opening `RetryJobModal` (see 6.4).
- Empty state: "No active jobs".

### 6.3 Job history — `/history`
- Full list of all jobs (including terminal), default sorted `created desc`.
- **Filters** (Mantine `MultiSelect`/`Select` + `TextInput`): status, category, free-text on title/hash. PB list API supports `filter`, `sort`, `perPage`, `page` — implement client-side pagination + server filter.
- Clicking a row opens a **detail Drawer**: full record fields, raw JSON, error stack, and (if error) a Restart button.
- **Clear completed** (optional): not required by issue; skip unless requested.

### 6.4 Restart failed job (Retry modal)
Invoked from Monitor or History on an `error` job.

- **Info panel**: shows torrent hash, category, original error.
- **Mode selector** (two-step, driven by job state):
  - *Retry from scratch* (default): re-run the normal pipeline. For anime → reset to `pending`; others → `pending_notify`.
  - *Skip directly to rsync* (only for `anime` category): requires manual input:
    - **Anime title** (`TextInput`, required)
    - **AnimeList ID** (`TextInput`, optional — left blank means "don't mark downloaded"; must be coordinated with backend behavior)
    - **TMDb ID / season** (read-only, pre-filled if known; editable optional)
- On submit → `POST /api/jobs/{id}/retry` with `{ mode: 'full' | 'rsync', anime_title?, anime_list_id?, tmdb_id?, tmdb_season? }`.
- After success: `Notifications.success`, modal closes, realtime pick up the updated status automatically (worker hooks enqueue on status change — no polling).

> **Backend dependency (out of frontend scope but required):** a `POST /api/jobs/{id}/retry` route that validates the job is in `error` state, then sets status to `pending_rsync` (anime, with filled `anime_title`/optional `anime_list_id`) or `pending_notify` (non-anime). Workers already pick these up via their hooks.

### 6.5 API playground — `/playground`
Ad-hoc HTTP client against the backend API (reuses superuser auth token automatically).

- **Method** select (GET/POST/PUT/DELETE).
- **Endpoint** select with presets + free-text path:
  - `GET /api/torrent-complete`
  - `POST /api/resolve-anime-title`
  - `POST /api/jobs/{id}/retry` (once added)
- **Params / JSON body** editor (`Textarea` with JSON validation; `folder_name`, `search_anime_list`, retry payload presets).
- **Send** → shows status code + pretty-printed JSON response in a `Code` block, plus request duration.
- Uses `pb.send()` so auth/CSRF headers are attached automatically.

## 7. Auth & security

- Superuser auth via PocketBase SDK; token persisted in `pb.authStore` (localStorage).
- All API calls go through the PB client so the `Authorization` header is attached.
- Route guard in `__root.tsx` beforeValidate: redirect to `/login` when unauthenticated.

## 8. Embedding in Go binary

**`ui/dist` is a build artifact and is NEVER committed to the repo.** The frontend is built in the GitHub release workflow before `go build`, so the binary always embeds a freshly built bundle.

### Go embed (build-tag guarded)

Because local dev runs the server and frontend separately, `go run .` must compile **without** `ui/dist` existing. Use build tags:

`ui/embed.go` — release build only:
```go
//go:build release

package ui

import "embed"

//go:embed all:dist
var Dist embed.FS
```

`ui/embed_stub.go` — default (dev) build:
```go
//go:build !release

package ui

import "embed"

// Empty placeholder so the package compiles without ui/dist.
//go:embed stub
var Dist embed.FS
```
(`ui/stub` is a single tracked placeholder file, e.g. an empty `index.html`, kept outside `dist/`.)

- Dev: `go run .` → uses the stub (no dist needed); the UI is served by Vite.
- Release: `go build -tags release ...` → embeds the real `ui/dist`.
- Serving (`serve_ui.go`): `se.Router.GET("/{path...}", apis.Static(fsys, true))` — PocketBase's SPA static helper serves hashed assets directly and falls back to `index.html` for unknown paths, so client-side routes work on refresh.
- `/api/*` and `/_/*` (PocketBase admin) are more specific ServeMux patterns and are never shadowed.

### Release workflow (`.github/workflows/release.yml`)

Insert a frontend build step before "Build binary":

```yaml
- name: Set up Node
  uses: actions/setup-node@v4
  with:
    node-version: 22
    cache: npm
    cache-dependency-path: ui/package-lock.json

- name: Build frontend
  working-directory: ui
  run: |
    npm ci
    npm run build
```

and change the Go build to `go build -tags release ...`.

## 9. Local development

Server and frontend run as two separate processes; the frontend points at the server via an env var.

```bash
# Terminal 1 — backend (PocketBase + API + workers), http://127.0.0.1:8090
go run .

# Terminal 2 — frontend dev server, http://localhost:5173 (Vite HMR)
cd ui
$env:VITE_PB_URL="http://127.0.0.1:8090"   # PowerShell; or put in ui/.env.local
npm install
npm run dev
```

- The PB SDK singleton reads the base URL from env: `new PocketBase(import.meta.env.VITE_PB_URL ?? '')` — empty string means "same origin", so the production embedded build needs no configuration.
- In dev, Vite serves the SPA on its own port and all PB/API calls go to `VITE_PB_URL`. `.env.local` is git-ignored so each dev can configure freely.

## 10. Implementation phases

1. **Scaffold**: re-scaffold `ui/` with Vite + React + TS; add Tanstack Router, Mantine, PB SDK deps; AppShell layout + theme; login + route guard. Configure `VITE_PB_URL`-aware PB client.
2. **Data layer**: PB client singleton, `types.ts`, `api.ts`, `useJobsRealtime` hook.
3. **Monitor page**: summary cards + live jobs table + status badges + realtime.
4. **Retry flow**: Retry modal + (with backend `POST /api/jobs/{id}/retry`) full & skip-to-rsync modes.
5. **History page**: filtered/paginated table + detail drawer.
6. **Playground**: endpoint presets + JSON editor + response viewer.
7. **Embed + CI**: `ui/embed.go` (release tag) + stub, SPA serving in `main.go`, frontend build step in `release.yml` + `-tags release`; delete outdated `web-ui.md`; add `ui/dist` to `.gitignore`.

## 11. Open questions / decisions

- `ui/dist` is never committed; CI builds it for releases, dev runs Vite separately. (Resolved per requirement.)
- Skip-to-rsync with empty `anime_list_id`: backend should treat empty as "skip MarkDownloaded" — confirm in backend implementation.
- Mantine color scheme: follow system (`auto`), brand color — pick a dark-friendly accent.
- Build tag name: `release` (default) vs `embed` — pick one and use consistently in `release.yml`.
