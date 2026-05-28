# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server (defaults to localhost:8080, SQLite at /tmp/aaru.db)
go run ./cmd/api

# Build
go build ./cmd/api
```

There are no tests, linters, or code generators configured.

## Architecture

Aaru is a Go + vanilla HTML/CSS/JS single-page application for managing deployment release pipelines across multiple environments. It uses Gin for HTTP routing, GORM with SQLite for persistence, and JWT for authentication.

### Layers (top to bottom)

```
cmd/api/main.go          — entry point, wires everything, starts server
internal/handler/        — Gin HTTP handlers (thin, delegate to services)
internal/middleware/      — JWT auth middleware (Bearer header or cookie)
internal/service/         — business logic (auth, RBAC, releases, blueprints, DMDB client)
internal/store/           — GORM/SQLite persistence (all CRUD)
internal/model/           — GORM models + config structs
web/                      — static frontend (templates/, js/, css/)
```

### Key architectural details

**DAG-based release pipeline** (`internal/service/release.go`, `internal/service/blueprint.go`): Promotable blueprints define a DAG of environment nodes and edges. When a release starts, source nodes (no incoming edges) become `in_progress`. A node only activates (`in_progress`) once ALL its parent nodes are `approved`. Sink nodes (no outgoing edges) being all approved marks the release `completed`. The DAG is validated using Kahn's algorithm (topological sort) on save to reject cycles.

**Three gate types per blueprint node** (`internal/model/types.go` → `BlueprintNode.GateType`):
- `manual` — human approval required; auto-creates an `approver-{env_code}` role with approve permissions
- `api_hook` — external system triggers promotion via `/api/hooks/promote/:stageId?token=xxx`; auto-generates a webhook token
- `auto` — auto-approves immediately when the stage activates (no human interaction)

**Release lifecycle**: `draft → in_progress → approved/rejected → completed/failed/rolled_back`. Individual stages follow `pending → in_progress → approved/rejected`. A single rejection fails the entire release.

**RBAC** (`internal/service/permission.go`): User → Role → Permission (deploy_unit_code + action). Actions: `deploy`, `approve`, `view`, `manage`. `*` wildcard means all deploy units. Seeded on first run with admin/developer/operator roles.

**Auth** (`internal/service/auth.go`, `internal/middleware/auth.go`): Mock GitLab SSO — login page shows configured mock users. JWT stored in cookie or Authorization header. `RequireAuth()` middleware for `/api/*`, `OptionalAuth()` available.

**Config** (`internal/model/config.go`): Looks for `./aaru.yaml`, then `~/.aaru/config.yaml`, then falls back to defaults (see README for the YAML schema).

**Frontend** (`web/`): Single JS file (`js/app.js`, ~740 lines) implementing SPA navigation, and a DAG editor using SVG + vanilla JS with drag, bezier-curve edges, and auto-layout. No bundler, no framework.

**DMDB integration** (`internal/service/dmdb.go`): External API at `dmdb.server_address` providing environment, silo, system, and deploy-unit data. Aaru proxies these through `/api/environments`, `/api/silos`, etc.

**DevOps API integration** (`internal/service/dmdb.go` → `ListAllDUs`, `CompareDUConfig`): Separate external API at `devops.server_address` (default `http://localhost:8733`) providing a unified DU list across all environments via `/api/v1/devops/list-du/`. The deploy units page uses this for the DU list (filterable by silo/system), then compares a selected DU's configuration across all DMDB environments via `/api/deploy-units/:code/compare`.
