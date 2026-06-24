---
name: score-cli
description: Query and mutate Santiment Score data (predictions, markets, portfolio, leaderboards, issuers, stakes) through the `score` CLI. Use when an agent or user needs to read or write Score product data from a shell with machine-readable JSON output.
---

## PURPOSE
Drive the Santiment Score product from the command line via the `score` binary,
which routes each command to the correct backend (Sanr or Arena) automatically
and returns raw JSON with stable exit codes for reliable automation.

You (the agent) are expected to **guide the user**, who may be non-technical:
acquire credentials for them, pick the right command, run it, and explain the
result. Never make the user hunt for flags or backends — that is your job.

## ACTIVATION
- The task involves Santiment Score data: predictions, signal markets, portfolio, leaderboards, competitions, pairs, prices, issuers, Hyperliquid stats, stakes, events, or contracts.
- The user asks to call the Sanr API (api.sanr.app) or Arena API (api-arena.santiment.net) from a shell.
- An agent needs scriptable, JSON output from the Score backends.

## DO NOT USE WHEN
- The `score` binary is not built/available — build it first with `task build`, then use `./bin/score`.
- The task targets a non-Score API or service.
- The task is pure data analysis on data already fetched (no API call needed).

## INPUTS
- A built `score` binary (`./bin/score` after `task build`, or on PATH).
- For authenticated or write commands: **one** Santiment Score API token (a Sanr
  JWT, ~5-year validity). The **same token authenticates both backends** — the CLI
  sends it as the Sanr bearer token and reuses it as the Arena `x-api-key`, so you
  configure it **once** and never per-backend. Save it with `score auth login`
  (preferred) or `SANR_TOKEN` / `--token`. There is no refresh flow. Public
  endpoints (health, markets, leaderboards, issuers/stakes/events/contracts lists)
  need no credentials.
- (optional) `--profile` / `--config` to select a credentials profile.

## GETTING THE API TOKEN (guide the user through this)
A command failed with exit 3 (auth), or you know it needs auth and no token is
configured (`score auth status --json` shows `"authorized": false`). Walk the
user through obtaining a token — do not give up or loop:

1. **Ask for their Sanr username** if you do not already know it (it is the
   handle on their Sanr profile, e.g. `agent_phoenix`).
2. **Give them this link**, substituting their username:
   `https://sanr.app/user/<username>/settings`
3. **Explain**: on that page open the **"Advanced"** section and click
   **"Generate token"** to create an API key, then paste the token back to you.
4. **Save it** once — this authenticates both backends and persists across runs:
   ```
   score auth login --token "<TOKEN>"
   ```
   (login tolerates a leading `Bearer ` or surrounding spaces.) For a one-off
   shell session you can instead `export SANR_TOKEN="<TOKEN>"`.
5. **Confirm** it took effect: `score auth status --json` → `"authorized": true`.

Notes:
- `score auth login` writes the token to the config file (default
  `~/.config/score/config.yaml`, mode 0600); every later command reads it
  automatically. No need to pass `--token` again or set `ARENA_API_KEY`.
- `score auth logout` clears the saved token.
- Treat the token as a secret: never echo it back in full, never log it, never
  put it in a shared file or a command the user did not authorize. Setup is
  one-time — the token is long-lived.

## PROCESS
1. Verify connectivity: `score health --json` (pings both backends; expect exit 0).
2. Ensure credentials if the command needs them. If `score auth status --json` shows `"authorized": false`, run the GETTING THE API TOKEN flow. A single `score auth login` covers both backends.
3. Discover the exact command/flags when unsure: `score describe --json` (full catalog) or `score <group> --help`. Do not guess flags.
4. Run the command with `--json`. Pass list filters via `--take/--skip/--sort/--filter` (Sanr) or `--limit/--offset` (Arena), and any other query parameter via repeatable `--param key=value`.
5. For write operations, build the JSON body from the schema (`score <command> --help` / `describe`) and pass it via `--data '<json>'` or `--data-file <path|->`. Confirm state-changing actions with the user first.
6. Check the exit code first, then parse stdout JSON. On non-zero, read the JSON error envelope on stderr and act per FAILURE HANDLING.

## DECISION RULES
- PRIORITY 1 — Backend routing is automatic; never pass a backend name. predictions/markets/portfolio/leaderboards/competitions/pairs/prices/profile → Sanr; issuers/stakes/events/contracts → Arena; `health` → both. Note `profile` is the authenticated user's own issuer record (Sanr); `issuers` is the public directory (Arena).
- PRIORITY 2 — Treat these subcommands as state-changing (require explicit user confirmation in interactive contexts, never run speculatively): `predictions create|update|close`, `portfolio order-place|order-cancel|deposit|withdrawal|distribution-save|recalc-historical`, `profile update|disable-notifications|disable-notifications-group|gdpr`. **`profile gdpr` deletes the user's account — never run it without explicit, unambiguous confirmation.**
- PRIORITY 3 — Prefer typed flags (`--take`, `--address`, `--symbols`) when present; fall back to `--param key=value` for any other query parameter. Values are parsed as JSON when possible (`--param generateImage=true` → boolean).
- PRIORITY 4 — When a command needs an id/username/address you do not have, list first to obtain a real one (e.g. `markets list` → use a `marketId`), then call the detail/write command.

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
score auth login --token "$TOKEN"     # save the token (authenticates Sanr + Arena), persists to config
score auth status --json              # check "authorized": true
score auth logout                     # remove the saved token
# Alternative for a one-off session instead of `login`:
export SANR_TOKEN="$TOKEN"
```

Write examples (state-changing — confirm first):
```
# predictions create requires marketId (from `markets list`) + direction (up|down):
score predictions create --data '{"marketId":"<marketId>","direction":"up"}' --json
score predictions update 486 --data '{"takeProfitPrice":70000}' --json
score predictions close 486 --json
score profile update --data '{"closedNoteHint":true}' --json
# Bodies whose exact shape you are unsure of: check `score <command> --help` /
# `score describe --json`, or have the user capture the payload from the web
# client's network tab. Pass via stdin with --data-file -:
echo '{"...":"..."}' | score portfolio deposit --data-file - --json
```

## OUTPUT REQUIREMENTS
- With `--json`: stdout is the raw API response body (unmodified), one payload, exit 0 on success.
- Errors: nothing on stdout; stderr carries `{"error":{"message","httpStatus","code","backend"}}`.
- Exit codes (stable contract): 0 success · 1 generic · 2 bad usage/flags · 3 auth (401/403) · 4 not found (404) · 5 rate limited (429) · 6 server error (5xx) · 7 network/timeout.
- Always branch on the exit code before parsing stdout.

## FAILURE HANDLING
- Exit 3 (auth): no token, or it is expired/revoked → run the GETTING THE API TOKEN flow (ask username → `https://sanr.app/user/<username>/settings` → Advanced → Generate token → `score auth login --token <TOKEN>`). One token covers both backends. Do not retry blindly.
- Exit 4 (not found): the id/username/address does not exist → re-list to find a valid identifier instead of retrying. Note: portfolio commands can return `ISSUER_NOT_FOUND` for an account that simply has no portfolio yet (e.g. a brand-new or GDPR-cleared account) — that is account state, not a CLI fault.
- Exit 7 (network/timeout): backend unreachable → run `score health --json` to localize the failure; raise `--timeout` for slow calls; retry with backoff a limited number of times.
- Exit 5 (rate limited): back off and retry after a short delay; do not hammer.
- Exit 2 (usage): a flag/arg is wrong → run `score <command> --help` or `score describe --json`; fix the invocation rather than retrying verbatim.

## EDGE CASES
- **No credentials yet (first run):** don't fail — start the GETTING THE API TOKEN flow. Ask for the username, hand over the settings link, explain Advanced → Generate token.
- **User doesn't know their username:** it's the handle in their Sanr profile URL (`sanr.app/user/<username>`). Ask them to open sanr.app while logged in and read it from the address bar.
- **Token pasted with noise:** `score auth login` already trims surrounding whitespace and a leading `Bearer `; still strip stray quotes from the user's paste.
- **Arena command returns 401/403:** the token isn't configured. Run `score auth login` (it covers Arena too); you do not need a separate `ARENA_API_KEY` unless Arena uses a different key.
- **`pairs favorites` returns 401 unauthenticated:** it requires the token even though other `pairs` subcommands are public.
- **`pairs aggregated` returns 400 `WRONG_SORT_OR_FILTER_FIELD`:** it needs a valid, non-empty `--filter` (filter JSON without outer braces, e.g. `--filter '"asset":"BTC"'`); an empty/unknown field is rejected.
- **Hyperliquid path:** the commands are `issuers hyperliquid <metrics|positions|trades|snapshots> <id>` — `issuers hyperliquid` alone just prints help, and there is no top-level `hyperliquid`/`issuers metrics`.
- **Empty but successful response:** some endpoints reply 200 with an empty body or empty `data` when there is simply no data — exit 0, not an error.
- **Detail/write command needs an id you don't have:** list first (`markets list`, `issuers list`, `predictions list`) and reuse a real id; don't invent one.
- **Unknown write-body schema:** never invent money amounts/asset prices. Read `score <command> --help` / `describe`, or ask the user to capture a real payload from the web client.
- **Binary missing / "command not found":** `task build`, then call `./bin/score` (or add it to PATH).
- **State-changing command requested casually:** confirm intent before running anything in PRIORITY 2, and treat `profile gdpr` (account deletion) as requiring explicit, repeated confirmation — prefer not to run it at all.
- **Human-readable vs JSON:** always pass `--json` when you will parse the output; human mode pretty-prints and is for people, not scripts.

## EXAMPLES
INPUT: "List the 3 most recent signal markets as JSON."
GOOD OUTPUT: `score markets list --take 3 --sort 'createdAt:desc' --json` → exit 0, JSON `{"data":[...]}` on stdout.
BAD OUTPUT: `score sanr markets --backend sanr` — invents a backend selector that does not exist; routing is automatic.

INPUT: "Show my portfolio balances." (no token configured)
GOOD OUTPUT: `score profile get --json` → exit 3 → start the token flow: ask the user's username, send them `https://sanr.app/user/<username>/settings`, explain Advanced → Generate token, then `score auth login --token "<TOKEN>"` and retry.
BAD OUTPUT: retrying `portfolio balances` repeatedly and reporting "auth error" without helping the user get a token.

INPUT: "Get the Hyperliquid metrics for issuer abc-123."
GOOD OUTPUT: `score issuers hyperliquid metrics abc-123 --json` → exit 0, metrics JSON.
BAD OUTPUT: `score hyperliquid abc-123` — wrong command path; the correct group is `issuers hyperliquid`.

INPUT: "Create an up prediction for BTC."
GOOD OUTPUT: `score markets list --param asset=BTC --take 1 --json` to find the `marketId`, confirm with the user, then `score predictions create --data '{"marketId":"<id>","direction":"up"}' --json`.
BAD OUTPUT: `score predictions create --data '{"symbol":"BTC/USD","direction":"up"}'` — `symbol` is not a field; the API requires `marketId`.
