#!/usr/bin/env bash
# Shared helpers for the tests-0.50.15 3-validator governance-upgrade test.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"
DATA_DIR="${ROOT_DIR}/data"

CHAIN_ID="${CHAIN_ID:-testing}"
DENOM="${DENOM:-orai}"
SEED_TAG="${SEED_TAG:-v0.50.13b}"   # EVM-era binary: mounts evm stores + resolves the Any
OLD_TAG="${OLD_TAG:-v0.50.14}"      # EVM soft-removed via fork; evm stores still mounted
PROPOSAL_ID="${PROPOSAL_ID:-316}"
UPGRADE_NAME="${UPGRADE_NAME:-v0.50.15}"

IMAGE_SEED="tests-0.50.15-seed:local"
IMAGE_OLD="tests-0.50.15-old:local"
IMAGE_NEW="tests-0.50.15-new:local"

NODES=(node1 node2 node3)
IPS=(172.28.0.11 172.28.0.12 172.28.0.13)
POWERS=(34 33 33)                       # voting power split (sum 100)
RPC_PORTS=(26657 26658 26659)           # host -> container 26657

log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  ok\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m  !!\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

compose() { ( cd "${ROOT_DIR}" && docker compose "$@" ); }

rpc_of() {  # rpc_of node2 -> http://127.0.0.1:26658
  local n="$1" i
  for i in "${!NODES[@]}"; do [[ "${NODES[$i]}" == "${n}" ]] && { echo "http://127.0.0.1:${RPC_PORTS[$i]}"; return; }; done
  die "unknown node ${n}"
}

# oraid one-shot in a throwaway container bound to a node's home (used pre-network).
oraid_at() {
  local home="$1" image="$2"; shift 2
  docker run --rm -v "${home}:/orai/.oraid" -e HOME=/orai "${image}" "$@" --home /orai/.oraid
}

# oraid inside a running compose node.
onode() {
  local n="$1"; shift
  compose exec -T "${n}" oraid "$@" --node tcp://127.0.0.1:26657
}

height_of() {
  local n="${1:-node1}"
  curl -sf "$(rpc_of "${n}")/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // empty' 2>/dev/null || true
}

# wait until node $1 reports height >= $2 (default now+2). returns 1 on timeout.
wait_height() {
  local n="${1:-node1}" target="${2:-}" timeout="${3:-90}" h=
  [[ -z "${target}" ]] && { h=$(height_of "${n}"); target=$(( ${h:-0} + 2 )); }
  log "waiting for ${n} height >= ${target} (timeout ${timeout}s)"
  for _ in $(seq 1 "${timeout}"); do
    h=$(height_of "${n}")
    [[ -n "${h}" && "${h}" -ge "${target}" ]] && { ok "${n} height=${h}"; return 0; }
    sleep 1
  done
  warn "${n} did not reach height ${target} within ${timeout}s (last=${h:-none})"
  return 1
}

image_elf_type() {
  docker run --rm --entrypoint readelf "$1" -h /usr/bin/oraid \
    | awk -F: '/Type:/ {gsub(/^[ \t]+/,"",$2); print $2}'
}

# query gov proposal ${PROPOSAL_ID} on node $1; sets Q_OUT / Q_RC.
query_proposal() {
  local n="${1:-node1}"
  set +e
  Q_OUT=$(compose exec -T "${n}" oraid query gov proposal "${PROPOSAL_ID}" \
            --node tcp://127.0.0.1:26657 -o json 2>&1)
  Q_RC=$?
  set -e
}

# write docker-compose.override.yml pinning every node to $1 (image ref).
pin_all_nodes() {
  local image="$1" f="${ROOT_DIR}/docker-compose.override.yml"
  {
    echo "services:"
    for n in "${NODES[@]}"; do
      printf '  %s:\n    image: %s\n' "${n}" "${image}"
    done
  } > "${f}"
}
