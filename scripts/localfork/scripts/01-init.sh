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

rm -rf "${DATA_DIR}"
mkdir -p "${DATA_DIR}/gentxs"

run() {
  local home_host="$1"; shift
  docker run --rm \
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
EOF

echo "✓ Initialized ${DATA_DIR} (A=${POWER_A}% B=${POWER_B}% S=${POWER_S}%, fork=${FORK_HEIGHT})"
