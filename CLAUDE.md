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

**One token, both backends.** Auth is a single long-lived Sanr JWT (~5y) accepted by both backends — sent as the Sanr `Authorization: Bearer` header and as the Arena `x-api-key` header. `config.Resolve` makes `ArenaAPIKey` **fall back to the resolved Sanr token**, so one credential covers everything; an explicit `--api-key`/`ARENA_API_KEY` still overrides. The single entry point is `score auth login --token <TOKEN>` (+ `status`/`logout`) — there is **no wallet-login and no refresh flow**. `config` holds only `SanrToken`/`ArenaAPIKey`, and `SaveTokens(path, profile, token)` caches that one token to `~/.config/score/config.yaml` (0600).

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

## Skill packaging for `npx skills` — read before bundling a binary

Skills are distributed with the Vercel `skills` CLI: `npx skills add santiment/skills`
copies each `skills/<name>/` directory (SKILL.md + sibling files) into the consumer's
agent skills dir (`.claude/skills/<name>/`, `.agents/skills/<name>/`, …). A skill must
be **self-contained**: an agent installs it into a foreign project that has no Go
toolchain, no Taskfile, no `./bin`. So a tool-backed skill ships its binary *inside the
skill*, not as a "build it first" instruction.

**The non-obvious transport constraint (this drove the whole design).** For a **public**
GitHub repo, `npx skills` does **not** git-clone by default — it fetches a text snapshot
from `skills.sh/api/download/...` and writes every file with `writeFile(contents, 'utf-8')`
(no base64 decode anywhere in its install path). **A raw committed binary is corrupted or
dropped on that path.** A git-clone fallback (which *would* preserve bytes) only kicks in
when the snapshot 404s — i.e. only until skills.sh indexes the repo. Don't rely on it.
Text files (scripts, base64) survive **both** paths intact.

**The pattern — bundle the binary as text.** Established by the `score` skill; copy it for
any new tool-backed skill (a skill is named after its tool, e.g. `skills/score/`):
- `task skill-bundle` cross-compiles the tool (`CGO_ENABLED=0`, pure `net/http` → trivial)
  for each `GOOS/GOARCH` in `SKILL_PLATFORMS` and writes `skills/<tool>/scripts/<tool>-<os>-<arch>.gz.b64`
  (gzip+base64 text, ~8 MB/platform). Default targets: `linux/amd64`, `darwin/arm64`.
- `skills/<tool>/scripts/run.sh` is the launcher the SKILL.md tells the agent to call
  (`bash <skill-dir>/scripts/run.sh <args>`). On first run it decodes the host's blob into
  `scripts/.bin/` (gitignored), `chmod +x`, and `exec`s it; later runs reuse the cache.
  Decode is base64→gunzip with a portable fallback (`base64 --decode` on GNU, else `openssl base64 -d`).
- `.gitattributes` pins `skills/<tool>/scripts/*.gz.b64 -text -diff` so CRLF normalization can't
  corrupt the encoded bytes.
- The SKILL.md states the tool ships bundled (no build step) and uses `score` as shorthand
  for the launcher. For an **unsupported platform** the launcher exits 7 with a message →
  add that `GOOS/GOARCH` to `SKILL_PLATFORMS` and rerun `task skill-bundle`.

Verify a skill end-to-end with `npx skills add ./skills/<name> --copy -a claude-code -y`
into a scratch dir, then run its `scripts/run.sh` from the installed location.
(Alternative if repo size from the blobs ever hurts: ship a `setup.sh` that downloads a
prebuilt binary from GitHub Releases instead of embedding it — needs a release workflow.)

## Conventions that matter when editing

- Agent-first UX is mandatory for any CLI here: `--json`, a `describe [--json]` catalog command, stable exit codes, stable flags, fully non-interactive (no hidden prompts). Mark state-changing subcommands as such in help and SKILL.md.
- Tests inject a buffer-backed printer before `Execute()`; `App.setup` only creates a printer when `app.printer == nil`. The routing tests in `internal/app/score/routing_test.go` spin httptest servers and assert which backend a command hits — mirror that when adding commands.
- Adding a tool/command or a new backend: follow `docs/conventions.md` (new `cmd/<tool>` + `internal/app/<tool>`, reuse `internal/platform`, vendor spec + add a `gen` step + `client.go`, add to `TOOLS` in `Taskfile.yml`, write `skills/<tool>/SKILL.md` from `skills/_template`). To make that skill installable + self-contained, follow **Skill packaging for `npx skills`** above (bundle the binary via `task skill-bundle` + a `scripts/run.sh` launcher).
