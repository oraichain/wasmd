# Localfork 3-validator hard-fork test (Docker)

## Goal

Reproduce recovery fork behavior:

| Node | Voting power | Binary before halt | Binary after halt | Expected |
|------|--------------|--------------------|-------------------|----------|
| **A** | 30% | **old** (`v0.50.13b`) | **new** (current + fork) | burn blacklist + continue |
| **B** | 30% | **old** (`v0.50.13b`) | **old** (`v0.50.13b`) | **apphash** / consensus failure |
| **S** | 40% | **old** (`v0.50.13b`) | **new** (current + fork) | continue with A (70% > 2/3) |

Halt sequence: stop **S** first (A+B=60% < 2/3 → chain halt) → stop **A** → restart **A+S** with new binary.

**Fork logic (new binary)** at `FORK_HEIGHT`:
1. Burn all `orai` on `BlacklistAddresses` (top balances from `genesis-balances.json`)
2. Transfer **all** CW20 of each `RecoveryFromAddress` on each contract in `RecoveryAssets` → `RecoveryAddress` via `ContractKeeper.Execute`
3. After fork block (`height > FORK_HEIGHT`): bank `SendRestriction` blocks in/out for those addresses

`RecoveryAssets` / `RecoveryFromAddress` are hardcoded (localfork Instantiate2 salts `localfork-cw20-v1` / `v2`). Only `RecoveryAddress` (tester) is baked via ldflags when rebuilding the new image.

Default mock `FORK_HEIGHT=40` (ldflags). Production default: `118018795`.

## Chain ID

Must be a known EVM chain id for `v0.50.13b`:
- mock default: **`testing`**
- also valid: `Oraichain`, `Oraichain-testnet`, `orai-1`

## Prerequisites

- Docker + Docker Compose
- `jq`, `curl`, `python3`
- Git tag `v0.50.13b` present locally
- `scripts/localfork/genesis-balances.json` (gitignored; required by `01-init.sh`)
- CW20 wasm (default `scripts/ibchooks/bytecode/cw20_base.wasm`; override with `CW20_WASM`)

## Quick start

```bash
cd scripts/localfork
chmod +x scripts/*.sh
FORK_HEIGHT=40 OLD_TAG=v0.50.13b ./scripts/run-all.sh
```

Or step by step:

```bash
export FORK_HEIGHT=40
export OLD_TAG=v0.50.13b
./scripts/00-build.sh old
./scripts/01-init.sh
./scripts/02-start-old.sh
./scripts/02b-deploy-cw20.sh   # store + Instantiate2 + mint → data/cw20.env
./scripts/03-wait-height.sh    # wait until FORK_HEIGHT-2
./scripts/04-halt.sh
set -a && source data/cw20.env && set +a
./scripts/00-build.sh new      # embeds FORK_HEIGHT + RecoveryAddress (= CW20_TO / tester)
./scripts/05-restart-new.sh
./scripts/06-verify.sh         # burn + bank send + CW20 rescue + B apphash
```

`run-all.sh` order: build old → init → start → **Instantiate2 CW20** → wait → halt → **rebuild new (`RecoveryAddress`)** → restart → verify.

## Genesis balances

Fresh local genesis (valset 30/30/40) + balances from `genesis-balances.json` (base units, 6 decimals).

Blacklist used by fork + verify: `blacklist-addresses.json` (must match `app/upgrades/v05014.BlacklistAddresses`).

Send-test account (not blacklisted): `test-address.json` — funded in genesis; `06-verify.sh` sends `tester → node-a` then `node-a → tester` after fork.

CW20 mock: fixed `cw20-deployer` key + `instantiate2` so contract addrs match `RecoveryAssets`; mint to first blacklist; at fork transfer full amount to `tester`. Config in `data/cw20.env`.

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

# burned blacklist balance on A (expect 0)
ADDR=orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w
curl -s "localhost:1317/cosmos/bank/v1beta1/balances/${ADDR}/by_denom?denom=orai" | jq .

# CW20 balances (after fork)
source data/cw20.env
docker exec -e HOME=/orai localfork-a oraid query wasm contract-state smart "$CW20_CONTRACT" \
  "{\"balance\":{\"address\":\"$CW20_FROM\"}}" --home /orai/.oraid --node tcp://127.0.0.1:26657

# B logs
docker logs localfork-b 2>&1 | grep -i apphash
```

## Cleanup

```bash
docker compose -f scripts/localfork/docker-compose.yml down -v
rm -rf scripts/localfork/data scripts/localfork/docker-compose.override.yml
```
