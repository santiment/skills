# san-skills

A monorepo for a **family of Go CLI utilities** and their companion `SKILL.md`
files. The CLIs are built for AI agents first: machine-readable `--json` output,
a self-describing `describe` command, stable exit codes, stable flags, and a
fully non-interactive mode.

The structure is a scaffold — new tools and skills are added without reshaping
the tree. See [`docs/conventions.md`](docs/conventions.md).

## Tools

| Tool | Description | Skill |
|------|-------------|-------|
| [`score`](cmd/score) | Agent-friendly client for the Santiment Score APIs (Sanr + Arena), with automatic backend routing | [`skills/score-cli`](skills/score-cli/SKILL.md) |

## Layout

```
cmd/<tool>/          entrypoints, one per CLI binary
internal/app/<tool>/ cobra command tree per CLI
internal/platform/   reusable packages: httpx, config, output, exitcode, apierr
internal/clients/    generated API clients (gen.go) + wiring (client.go)
api/                 vendored OpenAPI specs + oapi-codegen configs
tools/specnorm/      OpenAPI 3.1→3.0 normalizer
skills/              SKILL.md per tool
docs/                conventions and extension guide
```

## Requirements

- Go 1.26+
- [go-task](https://taskfile.dev) (`task`)
- `oapi-codegen` is pinned as a Go tool dependency (`go tool oapi-codegen`).

## Common tasks

```
task gen               # normalize specs + regenerate API clients
task build             # build all CLI binaries into ./bin
task test              # unit tests (no network)
task test-integration  # tests against live APIs (read-only)
task lint              # golangci-lint
```

## Quick start (`score`)

```
task build
./bin/score health --json            # ping both Score backends
./bin/score describe --json          # full command/flag/exit-code catalog
./bin/score markets list --take 5 --json
```

Credentials (only for authenticated endpoints) come from flags, environment
(`SANR_TOKEN`, `ARENA_API_KEY`, `SCORE_*`), or `~/.config/score/config.yaml`,
in that precedence. See the [score skill](skills/score-cli/SKILL.md) for the
full workflow and exit-code contract.
