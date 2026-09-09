# tests-0.50.15 — 3-validator governance upgrade to v0.50.15 (Docker)

Reproduces the full mainnet timeline on a **3-validator** local chain:

1. **EVM era** (`v0.50.13b`) — evm/feemarket/erc20/precisebank modules live; gov
   proposal **#316** carrying `/cosmos.evm.vm.v1.MsgUpdateParams` sits in state.
2. **Soft-remove fork** (`v0.50.14`) — EVM AppModules/keepers removed via the
   `v05014` BeginBlocker fork; the KV stores stay mounted (a fork can't trigger
   the upgrade store loader). `oraid query gov proposal 316` now errors
   server-side: `no concrete type registered for type URL
   /cosmos.evm.vm.v1.MsgUpdateParams`.
3. **Governance upgrade** (`v0.50.15`, `app/upgrades/v05015/upgrades.go`) —
   validators pass a `MsgSoftwareUpgrade`; at the plan height every `v0.50.14`
   node halts; operators swap in the new binary, which:
   - registers `evmlegacy.RegisterInterfaces` (decode-only stub) → proposal 316
     is queryable again;
   - drops the four EVM KV stores via `StoreUpgrades.Deleted` **and** stops
     mounting them in `app.go` (both are required — see below);
   - is built PIE by the Makefile `BUILD_MODE=pie` default.

## Images

| image | source | role |
|---|---|---|
| `tests-0.50.15-seed:local` | `git archive v0.50.13b` | EVM era — mounts evm stores, resolves the `Any`; used only to `InitChain` the seeded genesis |
| `tests-0.50.15-old:local` | `git archive v0.50.14` | pre-upgrade / "mainnet today" (`EXEC`, no stub, still mounts evm stores) |
| `tests-0.50.15-new:local` | current workspace, `make build` | post-upgrade (`DYN`/PIE, evmlegacy stub, v0.50.15 handler, evm stores deleted + unmounted) |

## Flow

```bash
cd scripts/tests-0.50.15
chmod +x scripts/*.sh
./scripts/run-all.sh
```

| step | what |
|---|---|
| `00-build.sh` | build all three images |
| `01-init.sh` | 3-validator genesis (34/33/33), short gov periods, splice proposal 316 — with the **v0.50.13b** binary |
| `02-seed-run.sh` | run all 3 **LIVE on v0.50.13b** for ~12 blocks; assert `query gov proposal 316` SUCCEEDS and the evm store is mounted. left running |
| `03-pre-upgrade.sh` | **genuine hot binary swap** v0.50.13b → v0.50.14 (no proposal): stop the live chain, start v0.50.14 on the same state; assert it resumes past H, reports the **same `app_hash(H)`** across the swap, `query gov proposal 316` now FAILS, evm store still mounted. left running |
| `04-gov-upgrade.sh` | submit `MsgSoftwareUpgrade` (`v0.50.15`, height H) → 3× vote → `PASSED` + plan scheduled → halt at H → swap all 3 to **new** → handler runs, evm stores pruned → `query gov proposal 316` works + PIE assert. left running |
| `05-restart-check.sh` | **cold-restart** all 3 on the new binary — proves the store deletion + `app.go` unmount are in sync (a mounted-but-deleted store panics `loadVersion` on every reboot) |
| `99-clean.sh` | tear down |

`02` env: `BLOCKS_ON_SEED` (12) — how long v0.50.13b runs before the swap.

## Why both `StoreUpgrades.Deleted` and the `app.go` unmount

`StoreUpgrades.Deleted` wipes the store data and removes it from `CommitInfo` at
the upgrade height — the chain runs fine immediately after. But
`rootmulti.loadVersion` runs on **every process start**, and a store key that is
still mounted (from `app.go`) yet absent from `CommitInfo` fails with:

```
version of store evm mismatch root store's version; expected N got 0
```

So declaring `Deleted` without removing the key from `NewKVStoreKeys(...)` gives a
chain that halts on the next validator restart. `05-restart-check.sh` is the
regression guard for this.

## While it runs

```bash
C="docker compose -f scripts/tests-0.50.15/docker-compose.yml"
$C ps
$C exec node2 oraid query gov proposal 316 --node tcp://127.0.0.1:26657 -o json
$C exec node3 oraid query upgrade applied v0.50.15 --node tcp://127.0.0.1:26657
```

RPC: node1 `:26657`, node2 `:26658`, node3 `:26659`.

Env overrides: `PROPOSAL_ID` (316), `UPGRADE_NAME` (v0.50.15), `SEED_TAG`
(v0.50.13b), `OLD_TAG` (v0.50.14), `CHAIN_ID` (testing), `DENOM` (orai).
