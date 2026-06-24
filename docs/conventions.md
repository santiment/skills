# Conventions

How this monorepo is organized and how to extend it without reshaping the tree.

## Repository layout

```
cmd/<tool>/          # main package per CLI binary (thin entrypoint)
internal/
  app/<tool>/        # cobra command tree for that CLI
  platform/          # reusable, CLI-agnostic packages (shared by all tools)
    httpx/           # HTTP engine: timeouts, retries, auth injection
    config/          # file + env + flag layering, profiles, login cache
    output/          # --json passthrough / human pretty-print, stderr errors
    exitcode/        # stable process exit codes + error→code mapping
    apierr/          # normalized API error type
  clients/<backend>/ # generated API client (gen.go) + hand-written wiring (client.go)
api/
  specs/             # vendored OpenAPI specs + *.normalized.json (generated)
  oapi-<backend>.yaml# oapi-codegen config per backend
tools/specnorm/      # OpenAPI 3.1→3.0 normalizer used before codegen
skills/<name>/       # SKILL.md per tool (see skills/README.md)
```

`internal/platform/*` is the contract every tool builds on. New tools reuse it
rather than re-implementing HTTP, config, or output behavior.

## Command naming

- `<tool> <resource> <verb>`, e.g. `score predictions list`, `score predictions create`.
- Resources are nouns (plural for collections); verbs are `list|get|create|update|close|...`.
- Backend selection is never a flag or argument — each command routes to the
  correct backend internally.
- List flags follow the backend's own convention: Sanr uses `--take/--skip/--sort/--filter/--attributes`; Arena uses `--limit/--offset/--sort/--filter`. Any other query parameter is passed via repeatable `--param key=value`.
- Request bodies for writes come from `--data '<json>'` or `--data-file <path|->`.

## Agent-friendly UX (required for every CLI)

- `--json`: machine-readable output (raw API body passthrough); errors as JSON on stderr.
- `describe`: a `describe [--json]` command emitting the full command/flag/exit-code catalog.
- Stable exit codes: 0 ok · 1 generic · 2 usage · 3 auth · 4 not found · 5 rate limited · 6 server · 7 network.
- Non-interactive: no hidden prompts; everything is flag/env/config driven.

## Adding a new CLI tool

1. Create `cmd/<tool>/main.go` calling `internal/app/<tool>.Execute()`.
2. Create `internal/app/<tool>/` with a cobra root and command files; reuse `internal/platform/*`.
3. If it talks to a new API, vendor its spec under `api/specs/`, add `api/oapi-<backend>.yaml`, add a `specnorm` + `oapi-codegen` step to `Taskfile.yml` (`gen`), and add a `client.go` wrapper under `internal/clients/<backend>/`.
4. Add the binary name to `TOOLS` in `Taskfile.yml`.
5. Write `skills/<tool>/SKILL.md` from `skills/_template/SKILL.md`.

## Code generation

- `task gen` runs `specnorm` (normalizes OpenAPI 3.1 → 3.0 and injects
  operationIds / parameter types) then `oapi-codegen` for each backend, then `go mod tidy`.
- `gen.go` is generated and never edited by hand; `client.go` is hand-written and
  never overwritten. Generated files carry the standard "DO NOT EDIT" header and
  are excluded from linting.

## Versioning

- Per-tool version lives in `internal/app/<tool>` (e.g. `score.Version`) and is
  overridable at build time with `-ldflags "-X .../<tool>.Version=vX.Y.Z"`.
- The exit-code contract and `--json` output shape are part of the public
  interface — keep them stable across releases.

## Testing

- `task test` — unit tests (httptest mocks, no network).
- `task test-integration` — tests tagged `//go:build integration` that hit live
  APIs (read-only/public endpoints; credentials via env for authed ones).
