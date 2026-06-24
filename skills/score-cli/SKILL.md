---
name: score-cli
description: Query and mutate Santiment Score data (predictions, markets, portfolio, leaderboards, issuers, stakes) through the `score` CLI. Use when an agent or user needs to read or write Score product data from a shell with machine-readable JSON output.
---

## PURPOSE
Drive the Santiment Score product from the command line via the `score` binary,
which routes each command to the correct backend (Sanr or Arena) automatically
and returns raw JSON with stable exit codes for reliable automation.

## ACTIVATION
- The task involves Santiment Score data: predictions, signal markets, portfolio, leaderboards, competitions, pairs, prices, issuers, Hyperliquid stats, stakes, events, or contracts.
- The user asks to call the Sanr API (api.sanr.app) or Arena API (api-arena.santiment.net) from a shell.
- An agent needs scriptable, JSON output from the Score backends.

## DO NOT USE WHEN
- The `score` binary is not built/available (build it first with `task build`).
- The task targets a non-Score API or service.
- The task is pure data analysis on data already fetched (no API call needed).

## INPUTS
- A built `score` binary (`./bin/score` after `task build`, or on PATH).
- For authenticated calls: a single long-lived Santiment Score API token (a Sanr JWT, ~5y validity). **The same token authenticates both backends** — it is sent as the Sanr bearer token and as the Arena `x-api-key`. Provide it via `--token` / `SANR_TOKEN` / `score auth set-token`; to authenticate Arena set `ARENA_API_KEY` (or `--api-key`) to the **same** value. There is no refresh flow. Public endpoints (health, markets, issuers list) need no credentials.
- (optional) `--profile` / `--config` to select a credentials profile.

## PROCESS
1. Verify connectivity: `score health --json` (pings both backends; expect exit 0).
2. Ensure credentials for the target command (see DECISION RULES). Set the token once via `score auth set-token --token <JWT>` or `SANR_TOKEN`; for Arena commands set `ARENA_API_KEY` to the same token value.
3. Discover the exact command/flags when unsure: `score describe --json` (full catalog) or `score <group> --help`.
4. Run the command with `--json`. Pass list filters via `--take/--skip/--sort/--filter` (Sanr) or `--limit/--offset` (Arena), and any other query parameter via repeatable `--param key=value`.
5. For write operations, pass the JSON body via `--data '<json>'` or `--data-file <path|->`.
6. Check the exit code first, then parse stdout JSON. On non-zero, read the JSON error envelope on stderr.

## DECISION RULES
- PRIORITY 1 — Backend routing is automatic; never pass a backend name. predictions/markets/portfolio/leaderboards/competitions/pairs/prices/profile → Sanr; issuers/stakes/events/contracts → Arena; `health` → both. Note `profile` is the authenticated user's own issuer record (Sanr); `issuers` is the public directory (Arena).
- PRIORITY 2 — Treat these subcommands as state-changing (require confirmation in interactive contexts, never run speculatively): `predictions create|update|close`, `portfolio order-place|order-cancel|deposit|withdrawal|distribution-save|recalc-historical`, `profile update|disable-notifications|disable-notifications-group|gdpr`.
- PRIORITY 3 — Prefer typed flags (`--take`, `--address`, `--symbols`) when present; fall back to `--param key=value` for any other query parameter. Values are parsed as JSON when possible (`--param generateImage=true` → boolean).

## TOOL USAGE
Global flags (apply to every command): `--json`, `--profile`, `--config`,
`--base-url-sanr`, `--base-url-arena`, `--token`, `--api-key`, `--timeout`,
`--verbose`, `-q/--quiet`.

Read examples:
```
score health --json
score markets list --take 5 --json
score markets list --param status=open --param asset=BTC --json
score predictions list --take 10 --sort 'createdAt:desc' --json
score predictions get 486 --json
score issuers list --limit 20 --json
score issuers hyperliquid metrics <issuerId> --json
score issuers hyperliquid positions <issuerId> --param status=open --json
score leaderboards players --take 10 --json
score profile get --json
score contracts list --json
```

Auth (one token for both backends; no refresh flow):
```
score auth set-token --token "$SANR_JWT"      # cache the token into the profile
export SANR_TOKEN="$SANR_JWT" ARENA_API_KEY="$SANR_JWT"   # same value for both backends
score auth status --json                       # show profile, base URLs, credential presence
```

Write examples (state-changing):
```
score predictions create --data '{"symbol":"BTC/USD","direction":"up"}' --json
score predictions close 486 --json
echo '{"amount":"100"}' | score portfolio deposit --data-file - --json
```

## OUTPUT REQUIREMENTS
- With `--json`: stdout is the raw API response body (unmodified), one payload, exit 0 on success.
- Errors: nothing on stdout; stderr carries `{"error":{"message","httpStatus","code","backend"}}`.
- Exit codes (stable contract): 0 success · 1 generic · 2 bad usage/flags · 3 auth (401/403) · 4 not found (404) · 5 rate limited (429) · 6 server error (5xx) · 7 network/timeout.
- Always branch on the exit code before parsing stdout.

## FAILURE HANDLING
- Exit 3 (auth): credentials missing or expired → set the token (`score auth set-token` or `SANR_TOKEN`, and `ARENA_API_KEY` to the same value for Arena) and retry; do not loop blindly.
- Exit 4 (not found): the id/username/address does not exist → re-list to find a valid identifier instead of retrying.
- Exit 7 (network/timeout): backend unreachable → run `score health --json` to localize the failure; raise `--timeout` for slow calls; retry with backoff a limited number of times.
- Exit 2 (usage): a flag/arg is wrong → run `score <command> --help` or `score describe --json`; fix the invocation rather than retrying verbatim.

## EXAMPLES
INPUT: "List the 3 most recent signal markets as JSON."
GOOD OUTPUT: `score markets list --take 3 --sort 'createdAt:desc' --json` → exit 0, JSON `{"data":[...]}` on stdout.
BAD OUTPUT: `score sanr markets --backend sanr` — invents a backend selector that does not exist; routing is automatic.

INPUT: "Get the Hyperliquid metrics for issuer abc-123."
GOOD OUTPUT: `score issuers hyperliquid metrics abc-123 --json` → exit 0, metrics JSON.
BAD OUTPUT: `score hyperliquid abc-123` — wrong command path; the correct group is `issuers hyperliquid`.
