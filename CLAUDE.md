# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A monorepo scaffold for a **family** of agent-friendly Go CLIs plus their `SKILL.md` files. One tool exists today: `score`, a client for the two Santiment Score backends. The structure is meant to grow new tools without reshaping the tree — see `docs/conventions.md` for the extension guide and `README.md` for the overview.

## Commands

```
task gen               # normalize OpenAPI specs + regenerate API clients (run after editing specs or specnorm)
task build             # build all CLI binaries into ./bin
task test              # unit tests, no network
task test-integration  # tests tagged //go:build integration; hit live Score APIs (read-only)
task lint              # golangci-lint (v2 config)
```

Run a single test: `go test ./internal/platform/config/ -run TestResolvePrecedence -v`
Run the binary directly: `go run ./cmd/score <args>` or `./bin/score <args>` after `task build`.

`oapi-codegen` is a pinned Go tool dependency (`go tool oapi-codegen`); no separate install needed. `task` and `golangci-lint` must be on PATH.

## Architecture — the non-obvious parts

**Two backends, automatic routing.** The `score` CLI talks to two independent APIs but never exposes a backend selector. Each command hard-codes which backend it uses:
- **Sanr** (`api.sanr.app`, JWT bearer): predictions, markets, portfolio, leaderboards, competitions, pairs, prices, profile (own `/v1/issuer`), auth.
- **Arena** (`api-arena.santiment.net`, x-api-key): issuers (+ Hyperliquid), stakes, events, contracts.
- `health` pings both.
Note `profile` (Sanr, the authenticated user's own record) and `issuers` (Arena, public directory) are deliberately separate commands because the resource name collides across backends with different schemas.

**One token, both backends.** Auth is a single long-lived Sanr JWT (~5y). The *same* token value is accepted by both backends — sent as the Sanr `Authorization: Bearer` header and as the Arena `x-api-key` header — so a single credential covers everything (set `ARENA_API_KEY` to the same value as `SANR_TOKEN`). There is **no refresh flow**: `config` holds only `SanrToken`/`ArenaAPIKey`, and `SaveTokens` caches just that one token.

**Codegen pipeline — read before touching API clients.** `task gen` does: `tools/specnorm` (normalize each vendored spec in `api/specs/*.json` → `*.normalized.json`) → `oapi-codegen` (per `api/oapi-{sanr,arena}.yaml`) → `go mod tidy`.
- `internal/clients/{sanr,arena}/gen.go` is **generated — never hand-edit** (carries the DO-NOT-EDIT header, excluded from lint, committed to git). `*.normalized.json` is generated and **gitignored**.
- `client.go` in those packages is **hand-written and never overwritten** — it wires the generated client to `httpx` and injects auth.
- **When codegen fails on a spec quirk, fix `tools/specnorm`, not `gen.go`.** Sanr is OpenAPI 3.1, which `oapi-codegen` (kin-openapi, 3.0-oriented) cannot fully consume. `specnorm` downgrades to 3.0 and rewrites: `type:[X,"null"]` → nullable, multi-type arrays → free-form, `oneOf:[ref,{type:"null"}]` nullable-union idiom, missing `operationId`s (derived from method+path), and untyped query parameters (inferred from `example`; untyped params otherwise generate `interface{}` fields that panic at runtime when nil).

**Command implementation pattern.** Every cobra command (`internal/app/score/*.go`) follows the same shape; copy an existing one when adding commands:
1. `client, _ := app.sanr()` or `app.arena()`.
2. Build the typed `*Params` via `queryFlags.into(&params)` — a JSON round-trip into the generated struct (its json tags match). Use `bindList(fs, style, defaultPage)` for list endpoints; `styleSanr` uses `--take/--skip/--attributes`, `styleArena` uses `--limit/--offset`. Both add `--param key=value` (JSON-typed) as an escape hatch for endpoint-specific params.
3. Request bodies for writes come from `bodyFlags.into(cmd, &body)` (`--data` / `--data-file <path|->`).
4. Emit with `app.emit(backend, resp.StatusCode(), resp.Body)` — the generated `*WithResponse` `.Body` is passed through raw (exact JSON for agents) and the status maps to a stable exit code. Transport errors go through `app.fail(err)`. Both record `app.exitCode` and return nil; `Execute()` returns that code.

**Reusable platform (`internal/platform/`)** — every future tool builds on this, not on its own HTTP/config/output code:
- `httpx`: the HTTP engine; implements the `Do(*http.Request)` interface the generated clients expect; retries 429/5xx with backoff.
- `config`: layers flags > env (`SANR_TOKEN`, `ARENA_API_KEY`, `SCORE_*`) > file profile (`~/.config/score/config.yaml`) > defaults; `SaveTokens(path, profile, token)` caches the single API token (the login cache).
- `output`: `--json` passes the API body through verbatim on stdout; human mode pretty-prints; errors are JSON `{"error":{...}}` on stderr.
- `exitcode` + `apierr`: the **stable exit-code contract** (0 ok, 1 generic, 2 usage, 3 auth, 4 not found, 5 rate-limited, 6 server, 7 network). This and the `--json` output shape are part of the public interface — keep them stable. An error can override its code via the `exitcode.Coder` interface (used for usage errors → 2).

## Conventions that matter when editing

- Agent-first UX is mandatory for any CLI here: `--json`, a `describe [--json]` catalog command, stable exit codes, stable flags, fully non-interactive (no hidden prompts). Mark state-changing subcommands as such in help and SKILL.md.
- Tests inject a buffer-backed printer before `Execute()`; `App.setup` only creates a printer when `app.printer == nil`. The routing tests in `internal/app/score/routing_test.go` spin httptest servers and assert which backend a command hits — mirror that when adding commands.
- Adding a tool/command or a new backend: follow `docs/conventions.md` (new `cmd/<tool>` + `internal/app/<tool>`, reuse `internal/platform`, vendor spec + add a `gen` step + `client.go`, add to `TOOLS` in `Taskfile.yml`, write `skills/<tool>/SKILL.md` from `skills/_template`).
