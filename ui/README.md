# qb-auto web UI

Frontend for [qb-auto](../README.md), built with Vite + React + TypeScript,
[TanStack Router](https://tanstack.com/router), [Mantine](https://mantine.dev)
and the [PocketBase JS SDK](https://github.com/pocketbase/js-sdk).

The production bundle is embedded into the Go binary via `go:embed` — see
`embed.go` / `embed_stub.go` and the `release` build tag.

## Development

Run the backend and frontend as two separate processes:

```bash
# Terminal 1 — backend (PocketBase + API + workers), http://127.0.0.1:8090
go run .   # from the repo root

# Terminal 2 — frontend dev server with HMR, http://localhost:5173
cd ui
$env:VITE_PB_URL="http://127.0.0.1:8090"   # PowerShell
# or: export VITE_PB_URL="http://127.0.0.1:8090"  # bash / put in ui/.env.local
npm install
npm run dev
```

`VITE_PB_URL` tells the app where the PocketBase server lives during
development. When empty (production/embedded build) the SDK uses the same
origin, so no configuration is needed.

## Scripts

| Command            | Description                          |
| ------------------ | ------------------------------------ |
| `npm run dev`      | Start the Vite dev server (HMR)      |
| `npm run build`    | Type-check and build to `dist/`      |
| `npm run lint`     | Run oxlint                           |
| `npm run preview`  | Preview the built `dist/` locally    |

## Layout

- `src/routes/` — page components (`/login`, `/`, `/history`, `/playground`)
- `src/router.tsx` — TanStack Router route tree + auth guard
- `src/components/` — AppShell layout, job table, retry modal, status badge
- `src/lib/` — PocketBase client, types, API wrappers, realtime hook

If you are developing a production application, we recommend enabling type-aware lint rules by installing `oxlint-tsgolint` and editing `.oxlintrc.json`:

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "plugins": ["react", "typescript", "oxc"],
  "options": {
    "typeAware": true
  },
  "rules": {
    "react/rules-of-hooks": "error",
    "react/only-export-components": ["warn", { "allowConstantExport": true }]
  }
}
```

See the [Oxlint rules documentation](https://oxc.rs/docs/guide/usage/linter/rules) for the full list of rules and categories.
