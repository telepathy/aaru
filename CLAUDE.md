# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server (defaults to localhost:8080, MySQL at 127.0.0.1:3306)
go run ./cmd/api

# Build (static binary, frontend embedded)
go build -o aaru ./cmd/api
```

There are no tests, linters, or code generators configured.

## Architecture

Aaru is a Go + vanilla HTML/CSS/JS single-page application for managing deployment release pipelines across multiple environments. It uses Gin for HTTP routing, GORM for persistence (MySQL), and JWT for authentication. Frontend assets are embedded into the binary via `//go:embed`.

### Layers (top to bottom)

```
cmd/api/main.go          — entry point, wires everything, starts server
internal/handler/        — Gin HTTP handlers (thin, delegate to services)
internal/middleware/      — JWT auth middleware (Bearer header or cookie)
internal/service/         — business logic (auth, RBAC, releases, blueprints, DMDB client)
internal/store/           — GORM persistence (all CRUD), MySQL only
internal/model/           — GORM models + config structs
web/                      — static frontend (templates/, js/, css/)
```

### Key architectural details

**DAG-based release pipeline** (`internal/service/release.go`, `internal/service/blueprint.go`): Promotable blueprints define a DAG of environment nodes and edges. When a release starts, source nodes (no incoming edges) become `in_progress`. A node only activates (`in_progress`) once ALL its parent nodes are `completed`. Sink nodes (no outgoing edges) being all completed marks the release `completed`. The DAG is validated using Kahn's algorithm (topological sort) on save to reject cycles.

**Three gate types per blueprint node** (`internal/model/types.go` → `BlueprintNode.GateType`):
- `manual` — human approval required; auto-creates an `approver-{env_code}` role with approve permissions
- `api_hook` — external system triggers promotion via `/api/hooks/promote/:stageId?token=xxx`; auto-generates a webhook token
- `auto` — auto-approves immediately when the stage activates (no human interaction)

**Release lifecycle**: `draft → in_progress → completed/failed/rolled_back`. Individual stages follow `pending → in_progress → approved → pushing → completed/rejected`. The `pushing` state indicates a DMDB config push is in progress; if it fails, the stage stays in `pushing` and can be retried via `retry-push`. A single rejection fails the entire release.

**Release creation wizard** (`web/js/app.js`): 4-step flow:
1. **选择DU与蓝图** — side-by-side selection: left panel picks a deploy unit (filterable by silo/system), right panel picks a promotion blueprint. Both must be selected to proceed.
2. **查看现状** — cross-environment comparison table showing only environments in the selected blueprint. Fields with differences across environments are highlighted.
3. **定义变更** — define field changes (ArtifactVersion required). Each field row shows only blueprint environments. Supports unified mode (same value for all envs) and per-env mode (different values per environment).
4. **预览** — per-environment preview of changes, filtered to blueprint environments only. Submits `{title, deploy_unit_code, blueprint_id, changes}` to `POST /api/releases`.

**DMDB batch update** (`internal/service/dmdb.go` → `UpdateDeployUnit`): Calls `POST /api/du-batch-update/{env}` on the DMDB service. The request body is a JSON array of update items, each containing `id`, `classCode`, and the fields to change. The response includes per-item status (`updated`, `not_found`, `forbidden`, etc.).

**DMDB model sync** (`internal/model/dmdb.go`): `DeployUnitInfo` mirrors the DMDB project's `DeployUnit` struct with all 37 fields and matching JSON tags (case-sensitive). Supporting types: `DatasourceAppConfig`, `RemoteServer`, `InitDbCfg`, `InitKafka`. `BatchUpdateResult` and `BatchUpdateResponse` model the batch update API response.

**RBAC** (`internal/service/permission.go`): User → Role → Permission (deploy_unit_code + action). Actions: `deploy`, `approve`, `view`, `manage`. `*` wildcard means all deploy units. Seeded on first run with admin/developer/operator roles.

**Auth** (`internal/service/auth.go`, `internal/middleware/auth.go`): Mock GitLab SSO — login page shows configured mock users. JWT stored in cookie or Authorization header. `RequireAuth()` middleware for `/api/*`.

**Config** (`internal/model/config.go`): YAML file (`./aaru.yaml` or `~/.aaru/config.yaml`) with environment variable overrides. Env vars take precedence. See README.md for the full list.

**Database** (`internal/store/db.go`): GORM-based, MySQL only. DSN from `AARU_DSN` env var. Auto-migrates all models on startup.

**Frontend** (`web/`): Single JS file (`js/app.js`, ~1300 lines) implementing SPA navigation, and a DAG editor using SVG + vanilla JS with drag, bezier-curve edges, and auto-layout. No bundler, no framework.

**DMDB integration** (`internal/service/dmdb.go`): External API at `dmdb.server_address` providing environment, silo, system, and deploy-unit data. Aaru proxies these through `/api/environments`, `/api/silos`, etc.

**DevOps API integration** (`internal/service/dmdb.go` → `ListAllDUs`, `CompareDUConfig`): Separate external API at `devops.server_address` (default `http://localhost:8733`) providing a unified DU list across all environments via `/api/v1/devops/list-du/`. The deploy units page uses this for the DU list (filterable by silo/system), then compares a selected DU's configuration across all DMDB environments via `/api/deploy-units/:code/compare`.

**DU cross-environment comparison** (`internal/service/dmdb.go` → `CompareDUConfig`): Fetches the full raw JSON for a given DU from all DMDB environments in parallel, flattens nested objects/arrays to JSON strings. `collapseInitTagOnly` post-processes `InitDb`/`InitDbAuth`/`InitDbFinal` fields: if the only difference across environments is the git blob tag in the source URL (and the tag matches `ArtifactVersion`), the raw JSON is replaced with a summary note. The frontend filters snapshots to only blueprint environments before display.
