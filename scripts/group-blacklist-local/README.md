# Local test: x/group → txfees blacklist

Verified on local 3-validator testnet (`scripts/multinode-local-testnet.sh`, chain-id `testing`).

## Addresses (this run)

| Role | Value |
|---|---|
| Admin / member1 (`validator1`) | `orai1msl85k8lhe646slkash4sl7yjyt3xzps7lgra2` |
| Member2 (`validator2`) | `orai13v9vefkjgad9vp2zk8z6hf9evnqnhy303f02uz` |
| Blacklist target (`gnad-eth`) | `orai1zy3gw5hk7fgzwcyjk6dnmslfj523gw6qeuzlrx` |
| Group ID | `1` |
| Group policy | `orai1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsp4a86q` |

`x/txfees/types/keys.go` `FoundationGroupPolicyAddress` was patched to the policy above for this local run. Rebuild + restart after changing it.

## Env

```bash
export HOME1=$HOME/.oraid/validator1
export CHAIN_ID=testing
export NODE=tcp://127.0.0.1:26657
export FROM=validator1
export ADMIN=orai1msl85k8lhe646slkash4sl7yjyt3xzps7lgra2
export DIR=$PWD/scripts/group-blacklist-local
```

Pattern for every tx: **generate-only → sign → broadcast**. Always pass `--fees` on generate-only.

---

## 0) Bring up testnet

```bash
make install
bash scripts/multinode-local-testnet.sh
# nodes: ~/.oraid/validator{1,2,3}
# RPC validator1: tcp://127.0.0.1:26657
```

---

## 1) Create group (1 member) — already done

Inputs:

- `01-members.json`
- `02-decision-policy.json` (`threshold=1`, `voting_period=60s`)

```bash
oraid tx group create-group-with-policy \
  $ADMIN "foundation-group" "foundation-policy" \
  $DIR/01-members.json $DIR/02-decision-policy.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE --fees 2000orai --gas 300000 \
  --generate-only > $DIR/tx-01-create-group-unsigned.json

oraid tx sign $DIR/tx-01-create-group-unsigned.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE > $DIR/tx-01-create-group-signed.json

oraid tx broadcast $DIR/tx-01-create-group-signed.json \
  --home $HOME1 --node $NODE --broadcast-mode sync -o json
```

Verify:

```bash
oraid q group groups --node $NODE
oraid q group group-info 1 --node $NODE
oraid q group group-members 1 --node $NODE
oraid q group group-policies-by-group 1 --node $NODE
oraid q group group-policy-info orai1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsp4a86q --node $NODE
```

---

## 2) Optional: add member later

```bash
oraid tx group update-group-members 1 $DIR/03-update-members.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE --fees 2000orai --gas 300000 \
  --generate-only > $DIR/tx-02-update-members-unsigned.json

oraid tx sign $DIR/tx-02-update-members-unsigned.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE > $DIR/tx-02-update-members-signed.json

oraid tx broadcast $DIR/tx-02-update-members-signed.json \
  --home $HOME1 --node $NODE --broadcast-mode sync -o json
```

```bash
oraid q group group-members 1 --node $NODE
```

---

## 3) Submit proposal: MsgAddBlacklist

JSON: `04-proposal-add-blacklist.json`

```bash
oraid tx group submit-proposal $DIR/04-proposal-add-blacklist.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE --fees 2000orai --gas 400000 \
  --generate-only > $DIR/tx-03-submit-add-unsigned.json

oraid tx sign $DIR/tx-03-submit-add-unsigned.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE > $DIR/tx-03-submit-add-signed.json

oraid tx broadcast $DIR/tx-03-submit-add-signed.json \
  --home $HOME1 --node $NODE --broadcast-mode sync -o json
```

Get proposal id from tx events (`EventSubmitProposal.proposal_id`).

---

## 4) Vote + exec

**Important:** `[voter]` must be the **bech32 address**, not the key name.

```bash
PROPOSAL_ID=1   # replace if needed

oraid tx group vote $PROPOSAL_ID $ADMIN VOTE_OPTION_YES "ok" \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE --fees 2000orai --gas 300000 \
  --generate-only > $DIR/tx-04-vote-unsigned.json

oraid tx sign $DIR/tx-04-vote-unsigned.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE > $DIR/tx-04-vote-signed.json

oraid tx broadcast $DIR/tx-04-vote-signed.json \
  --home $HOME1 --node $NODE --broadcast-mode sync -o json

oraid tx group exec $PROPOSAL_ID \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE --fees 2000orai --gas 400000 \
  --generate-only > $DIR/tx-05-exec-unsigned.json

oraid tx sign $DIR/tx-05-exec-unsigned.json \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE > $DIR/tx-05-exec-signed.json

oraid tx broadcast $DIR/tx-05-exec-signed.json \
  --home $HOME1 --node $NODE --broadcast-mode sync -o json
```

Shortcut (1 member / threshold 1):

```bash
oraid tx group submit-proposal $DIR/04-proposal-add-blacklist.json \
  --exec try \
  --from $FROM --home $HOME1 --keyring-backend test \
  --chain-id $CHAIN_ID --node $NODE --fees 2000orai --gas 400000 -y
```

---

## 5) Query / verify (verified: blacklisted=true)

```bash
oraid q txfees check-blacklist orai1zy3gw5hk7fgzwcyjk6dnmslfj523gw6qeuzlrx --node $NODE
oraid q txfees blacklist --node $NODE

oraid q group proposals-by-group-policy orai1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsp4a86q --node $NODE
oraid q group group-members 1 --node $NODE
oraid q group group-policy-info orai1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsp4a86q --node $NODE
```

---

## 6) Remove blacklist

JSON: `05-proposal-remove-blacklist.json` — same submit → vote → exec → query flow.
