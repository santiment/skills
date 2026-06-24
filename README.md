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
| [`score`](cmd/score) | Agent-friendly client for the Santiment Score APIs (Sanr + Arena), with automatic backend routing | [`skills/score`](skills/score/SKILL.md) |

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
task skill-bundle      # bundle the score binary (gzip+base64) into the score skill
task test              # unit tests (no network)
task test-integration  # tests against live APIs (reads by default; writes gated by env flags)
task lint              # golangci-lint
```

## Install as an agent skill (`npx skills`)

The `score` skill is **self-contained**: it bundles the `score` binary as a
gzip+base64 text blob, so installing the skill is all an agent needs — no Go, no
`task build`, no PATH setup.

```
npx skills add santiment/skills
```

This installs `skills/score` (SKILL.md + `scripts/`) into your agent's skills
directory. On first use the bundled `scripts/run.sh` launcher materializes the
right binary for your OS/arch (cached under `scripts/.bin/`) and runs it:

```
bash <skill-dir>/scripts/run.sh health --json
```

Default bundled targets are `linux/amd64` and `darwin/arm64`; rebuild the blobs
with `task skill-bundle` (extend `SKILL_PLATFORMS` in `Taskfile.yml` for more).

## Quick start (`score`, from this repo)

```
task build
./bin/score health --json            # ping both Score backends
./bin/score describe --json          # full command/flag/exit-code catalog
./bin/score markets list --take 5 --json
```

Authentication is a single long-lived Score API token (a Sanr JWT) that
**authenticates both backends** — the CLI sends it as the Sanr bearer token and
reuses it as the Arena `x-api-key`. Save it once:

```
./bin/score auth login --token "<TOKEN>"   # generate one at sanr.app → settings → Advanced → Generate token
./bin/score auth status --json             # "authorized": true
```

It is cached in `~/.config/score/config.yaml` (mode 0600) and reused by every
later command. Credentials otherwise resolve from flags > env (`SANR_TOKEN`,
`ARENA_API_KEY`, `SCORE_*`) > config file. There is no refresh flow, and no need
to set `ARENA_API_KEY` separately (it falls back to the token). See the
[score skill](skills/score/SKILL.md) for the full workflow and exit-code
contract.
