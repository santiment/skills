---
name: hyperhandler
description: Execute trading signals and monitor positions, orders, and balances on the Hyperliquid DEX through the `hyperhandler` CLI. Use when an agent or user needs to place/cancel orders, read account state, or manage HD wallets/keys for Hyperliquid from a shell with machine-readable JSON output.
---

## PURPOSE
Drive trading and monitoring on the Hyperliquid DEX from the command line via the
`hyperhandler` binary. It signs orders (EIP-712 + msgpack), places/cancels orders
from a JSON "signal", reads positions/orders/account state, and manages HD wallets
and keys. Output is raw JSON with stable exit codes for reliable automation.

The `hyperhandler` binary **ships inside this skill** — no build or install step
is needed. Run it through the bundled launcher next to this file:
`bash <this-skill-dir>/scripts/run.sh <args>`. On first run the launcher
materializes the right binary for the host OS/arch from a bundled (gzip+base64)
blob, caches it under `scripts/.bin/`, and execs it. **Below, `hyperhandler` is
shorthand for that launcher** — substitute `bash <skill-dir>/scripts/run.sh`, or
set `alias hyperhandler='bash <skill-dir>/scripts/run.sh'` for the session.

The tool is a **stateless executor and monitor**: each command performs one
action and exits. It keeps no background process and persists no trade history —
strategy, sizing, and risk limits are the caller's responsibility. Always start
from `hyperhandler describe --json` to discover commands, flags, and exit codes.

## SAFETY — read before any state-changing command
- **Default network is `testnet`.** Use `--network testnet` (or omit) for safe,
  fake-money runs; use `hyperhandler faucet` to fund a testnet account.
- **Mainnet writes require `--confirm`.** `exec` (without `--dry-run`) and `cancel`
  on `--network mainnet` refuse to run without `--confirm` and exit `2` without
  sending anything. This is a deliberate guardrail for agent-driven use.
- **Always `--dry-run` first** on a new `exec` to see the validated signal before
  risking funds. The config `security` section (`max_leverage`,
  `max_position_size_usd`, `require_stop_loss`) is enforced as a hard pre-trade
  gate; a violation fails validation (exit `2`) before anything is signed.
- **Keys never go on the command line.** Provide them via env or the OS keyring
  only; `config set-key` / `wallet import` read the secret from **stdin**.

## ACTIVATION
- The task involves trading or monitoring on Hyperliquid: place/cancel orders,
  read positions/orders/balances, set up or use a wallet/key.
- An agent needs scriptable JSON output from Hyperliquid with stable exit codes.

## DO NOT USE WHEN
- The task targets a different exchange or a non-Hyperliquid API.
- The task is pure analysis of already-fetched data (no API call needed).

## INPUTS
- The bundled `hyperhandler` launcher (`scripts/run.sh`) — self-provisions on first run.
- **Network**: `--network testnet|mainnet` (default `testnet`, or `network:` in
  config / `HL_NETWORK`).
- **Key** (for signing/monitoring your own account): a secp256k1 private key, via
  `HL_TESTNET_PRIVATE_KEY` / `HL_MAINNET_PRIVATE_KEY` / `HL_PRIVATE_KEY`, or stored
  in the OS keyring with `hyperhandler config set-key`. Read-only queries
  (`status`/`positions`/`orders`) accept `--address` to inspect any account
  without a key.
- **Signal** (for `exec`/`validate`): a JSON object on stdin or via `--signal <file>`:
  ```json
  {"pair":"BTC","side":"long","order_type":"limit","entry_price":67500,
   "size":0.1,"leverage":5,"stop_loss":66000,"take_profit":70000}
  ```
  `order_type` is `market` or `limit`; `entry_price` is required for `limit`.

## COMMANDS (run `hyperhandler describe --json` for the full catalog)
Read-only: `status`, `positions`, `orders` (all accept `--address`); `validate`.
State-changing (★ require `--confirm` on mainnet): `exec`★, `cancel`★.
Wallet/keys: `config set-key|remove-key|show-address|check`,
`wallet generate|import|list|use|delete`. Testnet: `faucet`.

## OUTPUT
- Pass `--json` for compact machine-readable JSON on stdout (human mode prints the
  same data pretty-printed). Errors are JSON `{"error":{...}}` on stderr.
- `--verbose` adds a structured `slog` audit trail on stderr (action, pair, side,
  size, price, nonce, network) — never the key.

## EXIT CODES (stable contract)
`0` ok · `1` generic error · `2` bad flags/usage (incl. mainnet without `--confirm`,
or a `security`-limit violation) · `3` auth (no key / invalid signature) · `4` not
found (unknown asset/order) · `5` rate limited · `6` server error · `7` network/timeout.

## FAILURE HANDLING
- Exit `3` "no private key": configure one — `printf %s "$KEY" | hyperhandler config set-key --network testnet`, or export `HL_TESTNET_PRIVATE_KEY`.
- Exit `2` on `exec`/`cancel` mainnet: re-run with `--network mainnet --confirm` once you have confirmed intent with the user.
- Exit `2` from `validate`/`exec`: the signal violated a config `security` limit; report which rule and adjust the signal or config.
- Exit `4` unknown asset: check the `pair` symbol against the venue.

## EXAMPLES
```
# Discover the interface
hyperhandler describe --json

# Fund + inspect a testnet account
printf '%s' "$TESTNET_KEY" | hyperhandler config set-key --network testnet
hyperhandler faucet --network testnet --json
hyperhandler status --json

# Dry-run a signal, then execute on testnet
cat signal.json | hyperhandler exec --dry-run --json
cat signal.json | hyperhandler exec --network testnet --json

# Execute on mainnet (explicit confirmation required)
cat signal.json | hyperhandler exec --network mainnet --confirm --json

# Cancel all orders for a pair
hyperhandler cancel --pair BTC --network testnet --json

# Monitor any address read-only (no key needed)
hyperhandler positions --address 0xabc... --json
```
