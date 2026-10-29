# CRM web shell (Slice 2)

React + TypeScript + Vite SPA shell on the frozen SN-207 tokens
(`src/shared/ui/tokens.css`, copied verbatim from `prototype/styles.css`).
All HTTP goes through the client generated from `api/openapi/openapi.yaml`.

## Commands

Backend API (from the repository root):

```bash
CRM_ENV=development CRM_DEV_AUTH=1 go run ./cmd/crm-api
```

`CRM_ENV` defaults to `production`, where the dev auth stub is refused at
startup. Listen address: `CRM_API_ADDR` (default `127.0.0.1:8080`).

Frontend dev server (proxies `/api` and `/health` to the API):

```bash
npm --prefix web ci
VITE_DEV_AUTH_USER=dev-user-001 npm --prefix web run dev
```

`VITE_DEV_AUTH_USER` (synthetic `dev-user-NNN` only) is ignored by production builds.

Regenerate the contract outputs after editing `api/openapi/**`
(bundle, TypeScript types, Go server types; never edit generated files):

```bash
npm --prefix web run openapi:gen
```

Tests and build:

```bash
go test ./...
npm --prefix web run typecheck
npm --prefix web run lint
npm --prefix web test
npm --prefix web run build
```
