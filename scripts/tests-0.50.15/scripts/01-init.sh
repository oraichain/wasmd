#!/usr/bin/env bash
# 3-validator genesis (power 34/33/33) + splice a "historical" EVM gov proposal
# (/cosmos.evm.vm.v1.MsgUpdateParams, id=${PROPOSAL_ID}) into
# .app_state.gov.proposals, with short gov voting/deposit periods.
set -euo pipefail
source "$(dirname "$0")/lib.sh"

docker image inspect "${IMAGE_SEED}" >/dev/null 2>&1 || die "run 00-build.sh first (missing ${IMAGE_SEED})"

log "stop previous run + wipe ${DATA_DIR}"
compose down -v --remove-orphans 2>/dev/null || true
rm -rf "${DATA_DIR}" "${ROOT_DIR}/docker-compose.override.yml"
for n in "${NODES[@]}"; do mkdir -p "${DATA_DIR}/${n}"; done

LIQUID=100000000000

log "init homes + validator keys"
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"; home="${DATA_DIR}/${n}"
  oraid_at "${home}" "${IMAGE_SEED}" init "${n}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" >/dev/null 2>&1
  oraid_at "${home}" "${IMAGE_SEED}" keys add "${n}" --keyring-backend test --output json \
    > "${home}/${n}_key.json" 2>"${home}/${n}_key.stderr" || true
  addr=$(jq -r '.address // empty' "${home}/${n}_key.json" 2>/dev/null || true)
  [[ -z "${addr}" ]] && addr=$(jq -r '.address // empty' "${home}/${n}_key.stderr" 2>/dev/null || true)
  [[ "${addr}" == orai1* ]] || { cat "${home}/${n}_key".*; die "no address for ${n}"; }
  echo "${addr}" > "${DATA_DIR}/${n}.addr"
  ok "${n} ${addr}"
done

log "build shared genesis on node1"
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"; addr=$(cat "${DATA_DIR}/${n}.addr")
  total=$(( POWERS[$i] * 1000000 + LIQUID ))
  oraid_at "${DATA_DIR}/node1" "${IMAGE_SEED}" genesis add-genesis-account "${addr}" "${total}${DENOM}" >/dev/null
done

GEN="${DATA_DIR}/node1/config/genesis.json"

log "shorten gov voting / deposit periods"
jq '
  .app_state.gov.params.voting_period = "15s"
  | .app_state.gov.params.expedited_voting_period = "10s"
  | .app_state.gov.params.max_deposit_period = "15s"
  | .app_state.gov.params.min_deposit = [{"denom":"'"${DENOM}"'","amount":"1000000"}]
  | .app_state.gov.params.expedited_min_deposit = [{"denom":"'"${DENOM}"'","amount":"2000000"}]
' "${GEN}" > "${GEN}.tmp" && mv "${GEN}.tmp" "${GEN}"

log "generate + splice EVM gov proposal id=${PROPOSAL_ID}"
( cd "${REPO_ROOT}" && go run ./scripts/tests-0.50.15/seed "${PROPOSAL_ID}" ) > "${DATA_DIR}/proposal.json"
jq -e '.messages[0]["@type"] == "/cosmos.evm.vm.v1.MsgUpdateParams"' "${DATA_DIR}/proposal.json" >/dev/null \
  || die "seed proposal JSON malformed"
jq --slurpfile p "${DATA_DIR}/proposal.json" --arg pid "${PROPOSAL_ID}" '
  .app_state.gov.proposals = ((.app_state.gov.proposals // []) + $p)
  | .app_state.gov.starting_proposal_id = (($pid | tonumber) + 1 | tostring)
' "${GEN}" > "${GEN}.tmp" && mv "${GEN}.tmp" "${GEN}"

log "gentx for each validator"
cp "${GEN}" "${DATA_DIR}/genesis.with-accounts.json"
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"; home="${DATA_DIR}/${n}"; ip="${IPS[$i]}"; power=$(( POWERS[$i] * 1000000 ))
  cp "${DATA_DIR}/genesis.with-accounts.json" "${home}/config/genesis.json"
  pk=$(oraid_at "${home}" "${IMAGE_SEED}" tendermint show-validator | tr -d '\r')
  oraid_at "${home}" "${IMAGE_SEED}" genesis gentx "${n}" "${power}${DENOM}" \
    --keyring-backend test --chain-id "${CHAIN_ID}" --ip "${ip}" --p2p-port 26656 --pubkey "${pk}" >/dev/null 2>&1
  mkdir -p "${DATA_DIR}/gentxs"
  cp "${home}/config/gentx/"*.json "${DATA_DIR}/gentxs/${n}.json"
done

log "collect-gentxs on node1"
rm -rf "${DATA_DIR}/node1/config/gentx"; mkdir -p "${DATA_DIR}/node1/config/gentx"
cp "${DATA_DIR}/gentxs/"*.json "${DATA_DIR}/node1/config/gentx/"
oraid_at "${DATA_DIR}/node1" "${IMAGE_SEED}" genesis collect-gentxs >/dev/null 2>&1
oraid_at "${DATA_DIR}/node1" "${IMAGE_SEED}" genesis validate >/dev/null 2>&1 || warn "genesis validate reported issues (continuing)"

jq -e --arg pid "${PROPOSAL_ID}" 'any(.app_state.gov.proposals[]; .id == $pid)' "${GEN}" >/dev/null \
  || die "proposal ${PROPOSAL_ID} missing from final genesis"

log "distribute genesis + wire peers / configs"
PEERS=()
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"
  id=$(oraid_at "${DATA_DIR}/${n}" "${IMAGE_SEED}" tendermint show-node-id | tr -d '\r')
  PEERS+=("${id}@${IPS[$i]}:26656")
done
for i in "${!NODES[@]}"; do
  n="${NODES[$i]}"; home="${DATA_DIR}/${n}"
  if [[ "${GEN}" != "${home}/config/genesis.json" ]]; then
    cp "${GEN}" "${home}/config/genesis.json"
  fi
  peer_list=$(IFS=,; echo "${PEERS[*]}")
  python3 - "${home}/config" "${peer_list}" <<'PY'
import re, sys, pathlib
d = pathlib.Path(sys.argv[1]); peers = sys.argv[2]
c = (d/"config.toml").read_text()
c = c.replace('laddr = "tcp://127.0.0.1:26657"', 'laddr = "tcp://0.0.0.0:26657"')
c = c.replace('cors_allowed_origins = []', 'cors_allowed_origins = ["*"]')
c = c.replace('addr_book_strict = true', 'addr_book_strict = false')
c = c.replace('allow_duplicate_ip = false', 'allow_duplicate_ip = true')
c = re.sub(r'^persistent_peers = ".*"$', f'persistent_peers = "{peers}"', c, count=1, flags=re.M)
c = re.sub(r'^timeout_commit\s*=\s*".*"$', 'timeout_commit = "1s"', c, count=1, flags=re.M)
(d/"config.toml").write_text(c)
a = (d/"app.toml").read_text()
a = a.replace('address = "localhost:9090"', 'address = "0.0.0.0:9090"')
a = re.sub(r'(\[api\](?:.*\n)*?enable = )false', r'\1true', a, count=1)
(d/"app.toml").write_text(a)
PY
done

cat > "${DATA_DIR}/meta.env" <<EOF
CHAIN_ID=${CHAIN_ID}
PROPOSAL_ID=${PROPOSAL_ID}
UPGRADE_NAME=${UPGRADE_NAME}
EOF

ok "3-validator genesis ready (34/33/33); gov proposal ${PROPOSAL_ID} carries /cosmos.evm.vm.v1.MsgUpdateParams"
