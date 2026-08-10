#!/usr/bin/env bash
# Initialize 3-validator localfork genesis:
#   node-a / node-b = 30% voting power each
#   node-s          = 40% voting power
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DATA_DIR="${ROOT_DIR}/data"
CHAIN_ID="${CHAIN_ID:-testing}"
DENOM="${DENOM:-orai}"
KEYRING=test
IMAGE="${INIT_IMAGE:-localfork-old:local}"
FORK_HEIGHT="${FORK_HEIGHT:-20}"

POWER_A=30
POWER_B=30
POWER_S=40
TOKENS_A=$((POWER_A * 1000000))
TOKENS_B=$((POWER_B * 1000000))
TOKENS_S=$((POWER_S * 1000000))
LIQUID=100000000000

NODES=(node-a node-b node-s)
IPS=(192.168.10.2 192.168.10.3 192.168.10.4)
POWERS=("$TOKENS_A" "$TOKENS_B" "$TOKENS_S")

# Stop any running localfork containers first — Docker Desktop cannot recreate
# bind-mount dirs that are still mounted by active containers.
if [[ -f "${ROOT_DIR}/docker-compose.yml" ]]; then
  (cd "${ROOT_DIR}" && docker compose down --remove-orphans 2>/dev/null) || true
fi

rm -rf "${DATA_DIR}"
mkdir -p "${DATA_DIR}/gentxs"
# Ensure host paths exist before docker bind-mount (Docker Desktop host_mnt quirk).
for n in "${NODES[@]}"; do
  mkdir -p "${DATA_DIR}/${n}"
done

run() {
  local home_host="$1"; shift
  mkdir -p "${home_host}"
  docker run --rm \
    -v "${home_host}:/orai/.oraid" \
    -e HOME=/orai \
    "${IMAGE}" "$@" --home /orai/.oraid
}

# Like run(), but attach stdin (needed for keys add --recover).
run_i() {
  local home_host="$1"; shift
  mkdir -p "${home_host}"
  docker run --rm -i \
    -v "${home_host}:/orai/.oraid" \
    -e HOME=/orai \
    "${IMAGE}" "$@" --home /orai/.oraid
}

echo "==> Init homes + keys"
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"
  mkdir -p "${DATA_DIR}/${n}"
  run "${DATA_DIR}/${n}" init "${n}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}"
  run "${DATA_DIR}/${n}" keys add "${n}" --keyring-backend "${KEYRING}" --output json \
    > "${DATA_DIR}/${n}/${n}_key.json"
done

# Non-blacklist account used by 06-verify to prove post-fork bank send still works (out + back).
TESTER_KEY=tester
TESTER_FUND=50000000000
run "${DATA_DIR}/node-a" keys add "${TESTER_KEY}" --keyring-backend "${KEYRING}" --output json \
  > "${DATA_DIR}/node-a/${TESTER_KEY}_key.json"
TESTER_ADDR=$(jq -r .address "${DATA_DIR}/node-a/${TESTER_KEY}_key.json")
echo "${TESTER_ADDR}" > "${DATA_DIR}/tester.address"
jq -n --arg address "${TESTER_ADDR}" '{address:$address,key:"tester"}' \
  > "${ROOT_DIR}/test-address.json"

# Fixed deployer for Instantiate2 CW20 (must match upgrades_test.go recoveryCW20*).
CW20_DEPLOYER_KEY=cw20-deployer
CW20_DEPLOYER_MNEMONIC="${CW20_DEPLOYER_MNEMONIC:-abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about}"
CW20_DEPLOYER_FUND=50000000000
# mnemonic + empty BIP39 passphrase
printf '%s\n\n' "${CW20_DEPLOYER_MNEMONIC}" | run_i "${DATA_DIR}/node-a" keys add "${CW20_DEPLOYER_KEY}" \
  --recover --keyring-backend "${KEYRING}" --output json \
  > "${DATA_DIR}/node-a/${CW20_DEPLOYER_KEY}_key.json"
CW20_DEPLOYER_ADDR=$(jq -r .address "${DATA_DIR}/node-a/${CW20_DEPLOYER_KEY}_key.json")
EXPECTED_CW20_DEPLOYER="${EXPECTED_CW20_DEPLOYER:-orai19rl4cm2hmr8afy4kldpxz3fka4jguq0a0nm77x}"
if [[ "${CW20_DEPLOYER_ADDR}" != "${EXPECTED_CW20_DEPLOYER}" ]]; then
  echo "ERROR: recovered cw20-deployer ${CW20_DEPLOYER_ADDR} != ${EXPECTED_CW20_DEPLOYER}"
  echo "  (must match upgrades_test.go recoveryCW20DeployerAddr for Instantiate2)"
  exit 1
fi
echo "${CW20_DEPLOYER_ADDR}" > "${DATA_DIR}/cw20-deployer.address"
echo "  cw20-deployer=${CW20_DEPLOYER_ADDR}"

BALANCES_JSON="${BALANCES_JSON:-${ROOT_DIR}/genesis-balances.json}"

echo "==> Build shared genesis accounts on node-a"
# reset node-a genesis as template
run "${DATA_DIR}/node-a" init node-a --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" -o

for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"
  power="${POWERS[$i]}"
  ADDR=$(jq -r .address "${DATA_DIR}/${n}/${n}_key.json")
  total=$((power + LIQUID))
  echo "  add-genesis-account ${n} ${ADDR} ${total}${DENOM}"
  run "${DATA_DIR}/node-a" genesis add-genesis-account "${ADDR}" "${total}${DENOM}"
done

echo "  add-genesis-account ${TESTER_KEY} ${TESTER_ADDR} ${TESTER_FUND}${DENOM}"
run "${DATA_DIR}/node-a" genesis add-genesis-account "${TESTER_ADDR}" "${TESTER_FUND}${DENOM}"

echo "  add-genesis-account ${CW20_DEPLOYER_KEY} ${CW20_DEPLOYER_ADDR} ${CW20_DEPLOYER_FUND}${DENOM}"
run "${DATA_DIR}/node-a" genesis add-genesis-account "${CW20_DEPLOYER_ADDR}" "${CW20_DEPLOYER_FUND}${DENOM}"

REVERT_JSON="${REVERT_JSON:-${ROOT_DIR}/revert-addresses.json}"
if [[ -f "${REVERT_JSON}" ]]; then
  echo "==> Add revert-address genesis balances from ${REVERT_JSON} (genesis > revert amount)"
  while IFS=$'\t' read -r REV_ADDR REV_AMT REV_GEN; do
    [[ -z "${REV_ADDR}" ]] && continue
    if [[ "${REV_GEN}" -le "${REV_AMT}" ]]; then
      echo "ERROR: revert genesis ${REV_GEN} must be > amount ${REV_AMT} for ${REV_ADDR}"
      exit 1
    fi
    echo "  add-genesis-account revert ${REV_ADDR} genesis=${REV_GEN} keep=${REV_AMT}"
    run "${DATA_DIR}/node-a" genesis add-genesis-account "${REV_ADDR}" "${REV_GEN}${DENOM}"
  done < <(jq -r '.[] | [.address, .amount, .genesis] | @tsv' "${REVERT_JSON}")
else
  echo "WARN: ${REVERT_JSON} not found — skip revert-address genesis funding"
fi

if [[ ! -f "${BALANCES_JSON}" ]]; then
  echo "ERROR: genesis balances file not found: ${BALANCES_JSON}"
  exit 1
fi

echo "==> Add genesis balances from ${BALANCES_JSON}"
# One container run: avoid N docker startups for many accounts.
# Skip SDK module account addresses: add-genesis-account would create a BaseAccount
# there and InitGenesis then panics with "account is not a module account".
ADD_SCRIPT="${DATA_DIR}/add-genesis-balances.sh"
python3 - <<PY
import hashlib
import json
from pathlib import Path

# cosmos-sdk crypto.AddressHash(name) = sha256(name)[:20]
MODULE_NAMES = (
    "fee_collector",
    "distribution",
    "mint",
    "bonded_tokens_pool",
    "not_bonded_tokens_pool",
    "gov",
    "transfer",
    "wasm",
    "tokenfactory",
    "evm",
    "erc20",
    "precisebank",
    "ibc",
    "ibcfee",
    "crisis",
    "nft",
    "group",
    "authz",
    "feegrant",
    "consensus",
    "upgrade",
    "params",
    "slashing",
    "staking",
    "bank",
    "capability",
    "evidence",
    "packetforward",
    "icq",
    "interchainaccounts",
    "icahost",
    "icacontroller",
)

def bech32_encode(hrp: str, witprog: bytes) -> str:
    charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

    def polymod(values):
        gen = [0x3B6A57B2, 0x26508E6D, 0x1EA119FA, 0x3D4233DD, 0x2A1462B3]
        chk = 1
        for v in values:
            b = chk >> 25
            chk = ((chk & 0x1FFFFFF) << 5) ^ v
            for i in range(5):
                chk ^= gen[i] if ((b >> i) & 1) else 0
        return chk

    def hrp_expand(h):
        return [ord(x) >> 5 for x in h] + [0] + [ord(x) & 31 for x in h]

    def convertbits(data, frombits, tobits, pad=True):
        acc = 0
        bits = 0
        ret = []
        maxv = (1 << tobits) - 1
        for value in data:
            acc = (acc << frombits) | value
            bits += frombits
            while bits >= tobits:
                bits -= tobits
                ret.append((acc >> bits) & maxv)
        if pad and bits:
            ret.append((acc << (tobits - bits)) & maxv)
        return ret

    data = convertbits(witprog, 8, 5)
    values = hrp_expand(hrp) + data
    polymod_val = polymod(values + [0, 0, 0, 0, 0, 0]) ^ 1
    checksum = [(polymod_val >> 5 * (5 - i)) & 31 for i in range(6)]
    return hrp + "1" + "".join(charset[d] for d in data + checksum)

skip = {
    bech32_encode("orai", hashlib.sha256(n.encode()).digest()[:20]): n
    for n in MODULE_NAMES
}

revert_path = Path("${REVERT_JSON}")
revert_addrs = set()
if revert_path.is_file():
    for row in json.loads(revert_path.read_text()):
        addr = (row.get("address") or "").strip()
        if addr:
            revert_addrs.add(addr)

path = Path("${BALANCES_JSON}")
script = Path("${ADD_SCRIPT}")
default_denom = "${DENOM}"
data = json.loads(path.read_text())
balances = data.get("balances") or data
if isinstance(balances, dict):
    balances = balances.get("balances") or []

lines = ["#!/bin/bash", "set -euo pipefail", "HOME=/orai"]
n = 0
skipped = []
for row in balances:
    addr = (row.get("address") or "").strip()
    amt = str(row.get("amount") or "").strip()
    denom = (row.get("denom") or data.get("denom") or default_denom).strip()
    if not addr or not amt or int(amt) <= 0:
        continue
    if addr in skip:
        skipped.append(f"{skip[addr]} ({addr})")
        continue
    if addr in revert_addrs:
        skipped.append(f"revert ({addr})")
        continue
    lines.append(
        f'oraid genesis add-genesis-account "{addr}" "{amt}{denom}" --home /orai/.oraid'
    )
    n += 1
if n == 0:
    raise SystemExit("no balances found in genesis-balances.json")
lines.append(f'echo "added {n} genesis balances"')
script.write_text("\n".join(lines) + "\n")
script.chmod(0o755)
print(f"  prepared {n} add-genesis-account commands (decimals={data.get('decimals', '?')})")
if skipped:
    print(f"  skipped {len(skipped)} module account address(es): {', '.join(skipped)}")
PY
docker run --rm \
  -v "${DATA_DIR}/node-a:/orai/.oraid" \
  -v "${ADD_SCRIPT}:/tmp/add-genesis-balances.sh:ro" \
  -e HOME=/orai \
  --entrypoint bash \
  "${IMAGE}" \
  /tmp/add-genesis-balances.sh

# Push shared genesis to all nodes before gentx
for n in "${NODES[@]}"; do
  if [[ "${n}" != "node-a" ]]; then
    cp "${DATA_DIR}/node-a/config/genesis.json" "${DATA_DIR}/${n}/config/genesis.json"
  fi
done

echo "==> Create gentxs"
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"
  ip="${IPS[$i]}"
  power="${POWERS[$i]}"
  PUBKEY=$(run "${DATA_DIR}/${n}" tendermint show-validator | tr -d '\r')
  run "${DATA_DIR}/${n}" genesis gentx "${n}" "${power}${DENOM}" \
    --keyring-backend "${KEYRING}" \
    --chain-id "${CHAIN_ID}" \
    --moniker "${n}" \
    --ip "${ip}" \
    --p2p-port 26656 \
    --pubkey "${PUBKEY}"
  cp "${DATA_DIR}/${n}/config/gentx/"*.json "${DATA_DIR}/gentxs/${n}.json"
done

echo "==> Collect gentxs"
rm -rf "${DATA_DIR}/node-a/config/gentx"
mkdir -p "${DATA_DIR}/node-a/config/gentx"
cp "${DATA_DIR}/gentxs/"*.json "${DATA_DIR}/node-a/config/gentx/"
run "${DATA_DIR}/node-a" genesis collect-gentxs
run "${DATA_DIR}/node-a" genesis validate || run "${DATA_DIR}/node-a" genesis validate-genesis || true

GEN_FINAL="${DATA_DIR}/node-a/config/genesis.json"
jq '
  .app_state.gov.params.voting_period = "20s"
  | .app_state.gov.params.expedited_voting_period = "15s"
' "${GEN_FINAL}" > "${GEN_FINAL}.tmp" && mv "${GEN_FINAL}.tmp" "${GEN_FINAL}"

for n in "${NODES[@]}"; do
  if [[ "${n}" != "node-a" ]]; then
    cp "${GEN_FINAL}" "${DATA_DIR}/${n}/config/genesis.json"
  fi
done

PEER_A=$(run "${DATA_DIR}/node-a" tendermint show-node-id | tr -d '\r')@192.168.10.2:26656
PEER_B=$(run "${DATA_DIR}/node-b" tendermint show-node-id | tr -d '\r')@192.168.10.3:26656
PEER_S=$(run "${DATA_DIR}/node-s" tendermint show-node-id | tr -d '\r')@192.168.10.4:26656

configure_node() {
  local home="$1"
  local peers="$2"
  python3 - <<PY
import re
from pathlib import Path

p = Path("${home}/config/config.toml")
t = p.read_text()
t = t.replace('persistent_peers = ""', 'persistent_peers = "${peers}"')
t = t.replace("addr_book_strict = true", "addr_book_strict = false")
t = t.replace("allow_duplicate_ip = false", "allow_duplicate_ip = true")
t = t.replace('cors_allowed_origins = []', 'cors_allowed_origins = ["*"]')
t = t.replace('laddr = "tcp://127.0.0.1:26657"', 'laddr = "tcp://0.0.0.0:26657"')
# Replace existing timeout_commit only (do NOT insert a duplicate key).
t, n = re.subn(r'^timeout_commit\s*=\s*".*"$', 'timeout_commit = "2s"', t, count=1, flags=re.M)
if n == 0:
    raise SystemExit(f"timeout_commit not found in {p}")
p.write_text(t)

app = Path("${home}/config/app.toml")
at = app.read_text()
at = at.replace("[api]\nenable = false", "[api]\nenable = true", 1)
at = at.replace('address = "tcp://localhost:1317"', 'address = "tcp://0.0.0.0:1317"')
at = at.replace('address = "localhost:9090"', 'address = "0.0.0.0:9090"')
app.write_text(at)
PY
}

configure_node "${DATA_DIR}/node-a" "${PEER_B},${PEER_S}"
configure_node "${DATA_DIR}/node-b" "${PEER_A},${PEER_S}"
configure_node "${DATA_DIR}/node-s" "${PEER_A},${PEER_B}"

cat > "${DATA_DIR}/meta.env" <<EOF
CHAIN_ID=${CHAIN_ID}
FORK_HEIGHT=${FORK_HEIGHT}
DENOM=${DENOM}
POWER_A=${POWER_A}
POWER_B=${POWER_B}
POWER_S=${POWER_S}
BALANCES_JSON=${BALANCES_JSON}
REVERT_JSON=${REVERT_JSON}
TESTER_KEY=${TESTER_KEY}
TESTER_ADDR=${TESTER_ADDR}
EOF

echo "✓ Initialized ${DATA_DIR} (A=${POWER_A}% B=${POWER_B}% S=${POWER_S}%, fork=${FORK_HEIGHT})"
echo "  genesis balances: ${BALANCES_JSON}"
echo "  send-test address: ${TESTER_ADDR} (${TESTER_KEY})"
