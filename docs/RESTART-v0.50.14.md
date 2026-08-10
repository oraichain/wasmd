# Oraichain restart runbook — v0.50.14 (EVM removal + fork)

Audience: Oraichain validators and node operators.
Applies to: the halted mainnet at height **118018794**, resuming at **118018795**.

---

## 1. What this binary changes

| | |
|---|---|
| Halt height (last committed block) | `118018794` |
| Fork height (first block after restart) | `118018795` = `v05014.ForkHeight` |
| Chain-id | `Oraichain` (unchanged) |
| Upgrade plan | **none** — this is a hard fork at a fixed height, not an `x/upgrade` plan |

The EVM is removed from the state machine: the `evm`, `feemarket`, `erc20` and
`precisebank` modules and keepers are gone, so `MsgEthereumTx` no longer decodes and the
ICS-20 precompile at `0x0802` is unreachable. IBC transfer is back to vanilla ibc-go.
The EVM JSON-RPC server is forced off regardless of `app.toml`.

At block `118018795` the fork handler burns the blacklisted balances, writes the blacklist
to the txfees store, moves the recovery assets, and pauses the affected pools.

---

## 2. ⚠️ There is no safety net for running the wrong binary

**Read this before scheduling the restart.**

There is no upgrade plan and no compiled-in halt height. Nothing stops a node from
starting the old `v0.50.13b` binary and producing a conflicting chain.

Divergence begins at the **first block produced**, `118018795`, for two independent
reasons:

1. The removed modules wrote state every block (`x/feemarket` `BeginBlock` persists the
   base fee; `x/vm` has its own begin/end block state). With those modules gone, the app
   hash differs from the old binary from the very first block — this is not gated by the
   fork height.
2. The fork handler burns and moves funds at `118018795`.

Consequences:

- **Cosmovisor will NOT auto-switch.** No `x/upgrade` plan named `v0.50.14` is registered,
  so no `upgrade-info.json` is written and `cosmovisor/upgrades/v0.50.14/bin/oraid` is
  never used. **Every operator must replace the binary manually** under the
  `cosmovisor/current` symlink (or wherever their unit resolves `oraid`).
- A node left on the old binary will hit an app-hash mismatch at `118018795` and panic.
  Followers fail loudly; that is fine. **Validators** on the old binary cannot vote on the
  new chain — if ≥1/3 of voting power stays on the old binary, nothing finalises and the
  chain stays halted.
- **Syncing from genesis with this binary is not possible.** Replaying pre-fork history
  without the EVM modules diverges from the historical chain. New and archive nodes must
  use state-sync or a snapshot taken at ≥ `118018795`.

Mitigation is procedural, not technical: publish the binary checksum, confirm it on every
validator before starting, and take a state backup first (§3) so a bad restart is
recoverable.

---

## 3. Before you restart: back up state

The fork is irreversible once committed, and every failure path in the fork handler is a
**panic** — deliberately, so a partial fork can never be applied. If the fork handler
panics at `118018795`, every node crashes in the same place on every restart attempt and
the chain cannot advance without a new binary. The backup below is what lets you retry.

With the node **stopped**:

```bash
sudo systemctl stop orai.service          # must already be stopped; verify with: systemctl is-active orai.service

ORAI_HOME=${ORAI_HOME:-$HOME/.oraid}
BACKUP=/var/backups/oraid-118018794-$(date -u +%Y%m%dT%H%M%SZ)

sudo mkdir -p "$BACKUP"
sudo cp -a "$ORAI_HOME/data"   "$BACKUP/data"
sudo cp -a "$ORAI_HOME/config" "$BACKUP/config"
sudo du -sh "$BACKUP"
```

Verify the backup records the expected height before proceeding:

```bash
cat "$ORAI_HOME/data/priv_validator_state.json"   # "height": "118018794"
```

Keep the backup until the chain has produced several hundred blocks past the fork and you
have verified §6.

**Rollback** (only while the chain has not finalised past the fork):

```bash
sudo systemctl stop orai.service
sudo rm -rf "$ORAI_HOME/data" "$ORAI_HOME/config"
sudo cp -a "$BACKUP/data"   "$ORAI_HOME/data"
sudo cp -a "$BACKUP/config" "$ORAI_HOME/config"
sudo chown -R "$(stat -c '%U:%G' "$ORAI_HOME")" "$ORAI_HOME"
# then restore the OLD binary before starting
```

---

## 4. Install the binary

Build must be **without** the `localfork` tag. A `localfork`-tagged binary refuses to run
the fork on `Oraichain` at mainnet heights (`assertNotLocalForkOnMainnet`), and the
production fixtures only run on chain-id `Oraichain` (`isForkChain`).

```bash
make build                                # or use the team-published release artifact
sha256sum build/oraid                     # MUST match the checksum published by the team
./build/oraid version --long              # confirm commit hash
```

Install it where the running unit resolves `oraid`. For a cosmovisor setup:

```bash
sudo install -m 0755 build/oraid "$ORAI_HOME/cosmovisor/current/bin/oraid"
"$ORAI_HOME/cosmovisor/current/bin/oraid" version --long
```

Do **not** create a `cosmovisor/upgrades/v0.50.14/` directory — there is no upgrade plan
to trigger it, and it will not be used.

---

## 5. Coordinated restart

1. Confirm **every** participating validator reports the same `sha256sum` and
   `version --long` commit. Collect this in the comms channel before anyone starts.
2. Agree a start time. Start together — the chain resumes only once ≥2/3 of voting power
   is online on the new binary.
3. Start:
   ```bash
   sudo systemctl enable --now orai.service
   sudo journalctl -fu orai.service
   ```
4. Watch the logs for the fork banner at height `118018795`:
   ```
   ========== running v0.50.14 fork logic ==========
   ========== blacklist burn complete ==========
   ========== txfees blacklist enabled ==========
   ========== fork logic applied to state ==========
   ```
   A panic here (`fork logic: ...`) means the fork aborted and no state was written. Stop,
   report it, and do not restart until a corrected binary is distributed — restarting will
   hit the same panic.

---

## 6. Post-restart verification

```bash
# blocks advancing, not catching up
curl -s http://localhost:26657/status | jq '.result.sync_info | {latest_block_height, catching_up}'

# supply: the minted excess must be gone
curl -s 'http://localhost:1317/cosmos/bank/v1beta1/supply/by_denom?denom=orai'

# blacklisted balances must be zero
for A in orai1vyghw3r3567y2algruuflqw2hx05vt6k945wrq \
         orai1ycryq0mghafwfwr346d5qe2ce8zvmflns08khy \
         orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w; do
  curl -s "http://localhost:1317/cosmos/bank/v1beta1/balances/$A/by_denom?denom=orai"
done

# EVM JSON-RPC must not be listening
curl -s -m 3 http://localhost:8545 && echo "UNEXPECTED: json-rpc is up" || echo "OK: json-rpc closed"
```

An `eth_*` call must fail — the JSON-RPC server is disabled in code
(`disableEVMJSONRPC` in `cmd/wasmd/commands.go`), so the port is simply not bound.

---

## 7. Known consequences to communicate publicly

- **IBC channel-146 (Injective) is one-way dead.**
  `orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w` is the escrow for `transfer/channel-146`.
  It is blacklisted, so its balance is burned at the fork and outbound sends are blocked
  permanently. Inbound transfers from Injective will fail. Sends *to* Injective still
  succeed and will not be redeemable — announce this, and consider closing the channel.
  Affected legitimate users are refunded manually.
- **Blacklisted accounts can still receive.** Only outbound sends are blocked. Anything
  sent to a blacklisted address after the fork is frozen. Publish the list.
- **New nodes cannot sync from genesis.** Publish a snapshot taken past `118018795`.
- IBC light clients on counterparty chains may have exceeded their `trusting_period`
  during the ~1 week halt and may need updating or governance client substitution.

---

## 8. Reference

| Item | Location |
|---|---|
| Fork height, burn/blacklist/recovery fixtures | `app/upgrades/v05014/constants.go` |
| Fork handler | `app/upgrades/v05014/fork.go` |
| Bank send restriction (height-gated) | `app/bank.go` |
| Ante blacklist (signer + authz MsgExec) | `x/txfees/ante/ante.go` |
| EVM JSON-RPC kill switch | `cmd/wasmd/commands.go` (`disableEVMJSONRPC`) |
| EVM-unreachable regression tests | `app/evm_removed_test.go` |
