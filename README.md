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
| [`hyperhandler`](cmd/hyperhandler) | Stateless executor/monitor for the Hyperliquid DEX: manual-mode order exec, positions/orders/balances, HD wallet + OS-keyring keys, EIP-712 signing. Defaults to testnet; mainnet writes require `--confirm` | [`skills/hyperhandler`](skills/hyperhandler/SKILL.md) |

## Layout

```
cmd/<tool>/          entrypoints, one per CLI binary
internal/app/<tool>/ cobra command tree per CLI
internal/platform/   reusable packages: httpx, config, output, exitcode, apierr
internal/clients/    API clients: generated (gen.go) + wiring, or hand-written (e.g. hyperliquid, no OpenAPI spec)
internal/<tool>/     tool-specific domain packages (non-platform), e.g. internal/hyperhandler/{signer,wallet,order,...}
api/                 vendored OpenAPI specs + oapi-codegen configs (only for spec-backed tools)
tools/specnorm/      OpenAPI 3.1→3.0 normalizer
skills/              SKILL.md + bundled binary per tool
docs/                conventions and extension guide
```

## Requirements

- Go 1.26+
- [go-task](https://taskfile.dev) (`task`)
- `oapi-codegen` is pinned as a Go tool dependency (`go tool oapi-codegen`).

## Common tasks

```
task gen               # normalize specs + regenerate API clients (spec-backed tools only)
task build             # build all CLI binaries into ./bin
task skill-bundle      # bundle each tool's binary (gzip+base64) into its skill
task test              # unit tests (no network)
task test-integration  # tests against live APIs (score: reads; hyperhandler: live testnet; writes gated by env flags)
task lint              # golangci-lint
```

## Install as an agent skill (`npx skills`)

Each skill is **self-contained**: it bundles its tool's binary as a gzip+base64
text blob, so installing the skill is all an agent needs — no Go, no
`task build`, no PATH setup.

```
npx skills add santiment/skills
```

This installs the available skills (`score`, `hyperhandler`) — each a SKILL.md +
`scripts/` — into your agent's skills directory. On first use the bundled
`scripts/run.sh` launcher materializes the right binary for your OS/arch (cached
under `scripts/.bin/`) and runs it:

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

## Quick start (`hyperhandler`, from this repo)

```
task build
./bin/hyperhandler describe --json                       # full command/flag/exit-code catalog
./bin/hyperhandler --network testnet status --address 0x... --json   # read-only, no key needed
echo '{"pair":"BTC","side":"long","order_type":"market","size":0.1}' \
  | ./bin/hyperhandler exec --dry-run --json             # validate + price, sends nothing
```

`hyperhandler` defaults to **testnet**; a state-changing command on mainnet
(`exec` without `--dry-run`, `cancel`) refuses to run without `--confirm`. Keys
are read from env (`HL_TESTNET_PRIVATE_KEY` / `HL_MAINNET_PRIVATE_KEY` /
`HL_PRIVATE_KEY`) or the OS keyring (`hyperhandler config set-key`, reads the key
from stdin) — never from argv. Config lives at `~/.hyperhandler/config.yaml`. See
the [hyperhandler skill](skills/hyperhandler/SKILL.md) for the full workflow and
exit-code contract.
