# forkprod — dress rehearsal for the mainnet restart

## Why this exists

Mainnet is halted at height `118018794`. To restart it we hand every validator one binary,
they all start it, and at block `118018795` that binary burns the attacker's funds, freezes
their accounts, rescues the recoverable assets and pauses the affected pools.

We get **one attempt**. If anything in that fork step is wrong, every node dies at the same
block and the chain cannot move again until we build, test and redistribute a new binary.

So the question this harness answers is narrow and specific:

> If we hand validators *this exact binary*, does the chain come back — and does it come
> back with the right balances?

## Why it isn't the same as `scripts/localfork`

`localfork` already tests the *mechanism*: three validators, a halt, a binary swap, the
burn logic, and a stale validator forking off. It is fast and it stays useful.

But `localfork` does not test the binary we are going to ship. It is built with a special
`localfork` flag that swaps in a whole parallel set of test values — pretend wallets,
pretend amounts, a fork height of 40 instead of 118018795, pool pausing switched off
entirely. It proves the machinery works. It cannot tell you whether the **real numbers**
are right, because it never loads them.

That gap matters here. A mistyped address or a wrong burn amount in the real list is
invisible to `localfork` and fatal on restart day.

`forkprod` closes it. It builds the binary with no test flags at all, runs it on a chain
that calls itself `Oraichain`, starts a handful of blocks below the real fork height, and
lets it execute the genuine recovery list against genuine contract code.

In short:

- **localfork** — "does the fork machinery work?" Fast. Run it while developing.
- **forkprod** — "is the binary we're about to ship correct?" Slower. Run it before restart.

Keep both. Neither replaces the other.

## What it proves

The chain is built with three validators — **A 30%, B 30%, S 40%** — started on the current
production binary `v0.50.13b`, then stopped at exactly one block before the fork, the same
place mainnet is sitting now. **A and S** restart on the new binary. **B deliberately stays
on the old one.**

After the fork the harness checks:

1. The chain actually restarts and passes the fork height.
2. Every blacklisted account is emptied.
3. Every recovery-list account gives up exactly the intended amount and keeps the rest.
4. Rescued tokens leave the attacker's wallet and land on the recovery wallet.
5. Both pools end up paused.
6. The freeze list was written, and money can still be sent *to* a frozen account
   (where it then stays).
7. Ordinary users can still transact normally.
8. The EVM is genuinely gone — even with the old config file still asking for it, nothing
   answers.
9. **Node B, left on the old binary, forks off instead of quietly following.**

Point 9 is the one to show validators. It is not theoretical: the run prints the two
different fingerprints the two binaries produce for the same block. Anyone who starts the
wrong binary on restart day leaves the network — silently, from their own point of view.

There are two extra scenarios:

- **Wrong network** — the same binary pointed at any chain that isn't Oraichain refuses to
  apply the recovery list and just keeps producing blocks. Real money movements cannot
  leak onto a test network.
- **Bad input** — deliberately corrupt one number in the recovery list and confirm the
  chain stops loudly at the fork with nothing written, *and stays stopped on restart*.
  This is the disaster case, made visible on purpose: restarting does not fix it, only a
  corrected binary does.

## What it does not prove

**The freeze itself is not tested end to end.** Blocking outgoing transfers from a frozen
account requires signing as that account, and these are real attacker wallets — nobody has
those keys, and there is no way to add a test wallet to the freeze list. What the harness
checks is that the freeze list is written correctly and that incoming funds behave as
expected. The blocking behaviour is covered by the Go tests in
`app/upgrades/v05014` (`TestBlacklistRestrictionOnlyAfterForkBlock`).

**The account balances are invented.** Real mainnet balances are not used, so this does not
confirm that the recovery amounts match what the attacker actually holds. It confirms the
binary applies them correctly and consistently. Checking the amounts themselves against
mainnet is a separate exercise.

**The contracts are stand-ins.** Real contract code is used, but instances are created
fresh and given the production addresses, since the originals cannot be recreated on a test
network.

## Running it

```bash
cd scripts/forkprod
./scripts/run-all.sh
```

Everything runs as ordinary local processes — no Docker. The first run builds two binaries
and takes several minutes; later runs are quicker. Individual steps can be run on their own
in numbered order.

To clean up:

```bash
source scripts/env.sh && stop_all
rm -rf data
```

## Files

| | |
|---|---|
| `fixtures/` | reads the recovery list straight out of the compiled binary, so the test data can never drift from what ships |
| `scripts/00-build.sh` | builds the new binary and `v0.50.13b` |
| `scripts/01-bootstrap-contracts.sh` | creates the contracts the recovery step needs, at their production addresses |
| `scripts/02-init.sh` | builds the three-validator chain, starting just below the fork height |
| `scripts/03…05` | start on the old binary, halt, restart A+S on the new one |
| `scripts/06-verify.sh` | the checks listed above |
| `scripts/07`, `08` | the wrong-network and bad-input scenarios |
