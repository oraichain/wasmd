# Localfork 3-validator hard-fork test (Docker)

## Goal

Reproduce recovery fork behavior:

| Node | Voting power | Binary before halt | Binary after halt | Expected |
|------|--------------|--------------------|-------------------|----------|
| **A** | 30% | **old** (`v0.50.13b`) | **new** (current + fork) | continues, applies fork state |
| **B** | 30% | **old** (`v0.50.13b`) | **old** (`v0.50.13b`) | **apphash** / consensus failure |
| **S** | 40% | **old** (`v0.50.13b`) | **new** (current + fork) | continues with A (70% > 2/3) |

Halt sequence: stop **S** first (A+B=60% < 2/3 → chain halt) → stop **A** → restart **A+S** with new binary.

**Binaries**
- `localfork-old:local` ← `git archive` tag **`v0.50.13b`** (override with `OLD_TAG=...`)
- `localfork-new:local` ← current workspace (`FORK_HEIGHT` via ldflags)

`RunForkLogic` is currently a **noop** (mock mint/burn removed). After real fork logic is implemented, restore state-marker checks in `scripts/06-verify.sh` and re-run.

Default mock `FORK_HEIGHT=40` (via ldflags). Production default remains `118018800`.

## Chain ID

Must be a known EVM chain id for `v0.50.13b`:
- mock default: **`testing`**
- also valid: `Oraichain`, `Oraichain-testnet`, `orai-1`

Do **not** use arbitrary ids like `localfork-1` (old binary panics: `unknown chain id`).

- Docker + Docker Compose
- `jq`, `curl`, `python3`
- Git tag `v0.50.13b` present locally
- Enough disk/RAM to build `oraid` twice

## Quick start

```bash
cd scripts/localfork
chmod +x scripts/*.sh
FORK_HEIGHT=20 OLD_TAG=v0.50.13b ./scripts/run-all.sh
```

Or step by step:

```bash
export FORK_HEIGHT=20
export OLD_TAG=v0.50.13b
./scripts/00-build.sh      # old=v0.50.13b , new=HEAD+fork
./scripts/01-init.sh       # genesis A30/B30/S40 + genesis-balances.json
./scripts/02-start-old.sh  # all 3 on old binary
./scripts/03-wait-height.sh
./scripts/04-halt.sh       # stop S, then A
./scripts/05-restart-new.sh # A+S=new, B=old
./scripts/06-verify.sh     # fork marker + B apphash
```

This harness boots a **fresh local genesis** (exact 30/30/40 valset) and injects
wallet balances from `genesis-balances.json` — not a full mainnet state fork.

Amounts are **base units** (`orai`, 6 decimals). Example: `600003387` ORAI → `600003387000000`.

Override: `BALANCES_JSON=/path/to.json ./scripts/01-init.sh`

## Using a mainnet snapshot later

To drive the same flow from full snapshot state instead of balance-only genesis:

1. Sync/export on Ubuntu: `oraid export --for-zero-height > state.json`
2. Replace CometBFT/app valset with A/B/S keys + powers 30/30/40
3. Copy `state.json` → each `data/node-*/config/genesis.json` and copy `wasm/`
4. Reuse `02`→`06` scripts (set `FORK_HEIGHT` near snapshot height, e.g. `118018800`)

## Ports

| Node | RPC | LCD | gRPC |
|------|-----|-----|------|
| A | 26657 | 1317 | 9090 |
| B | 26667 | 1327 | 9100 |
| S | 26677 | 1337 | 9110 |

## Manual queries

```bash
# height
curl -s localhost:26657/status | jq .result.sync_info.latest_block_height

# B logs (apphash only after real fork state changes land)
docker logs localfork-b 2>&1 | grep -i apphash
```

## Cleanup

```bash
docker compose -f scripts/localfork/docker-compose.yml down -v
rm -rf scripts/localfork/data scripts/localfork/docker-compose.override.yml
```
