#!/usr/bin/env bash
set -euo pipefail

RPC="${RPC:-http://127.0.0.1:26657}"
TARGET="${1:-}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Halt buffer: block time ~2s; stop S/A takes a few seconds. Use H-5 (not H-2) so
# we cannot race past ForkHeight before consensus actually stops.
HALT_BUFFER="${HALT_BUFFER:-5}"

if [[ -z "${TARGET}" ]]; then
  if [[ -f "${ROOT_DIR}/data/meta.env" ]]; then
    # shellcheck disable=SC1091
    source "${ROOT_DIR}/data/meta.env"
    TARGET=$((FORK_HEIGHT - HALT_BUFFER))
  else
    TARGET=18
  fi
fi

if [[ "${TARGET}" -lt 2 ]]; then
  echo "ERROR: halt target ${TARGET} too low; raise FORK_HEIGHT (need >= HALT_BUFFER+2)"
  exit 1
fi

echo "==> Waiting until height == ${TARGET} (must halt BEFORE fork; buffer=${HALT_BUFFER})"
while true; do
  H=$(curl -sf "${RPC}/status" | jq -r '.result.sync_info.latest_block_height // empty' || true)
  if [[ -n "${H}" ]]; then
    if [[ "${H}" -ge "${TARGET}" && "${H}" -lt "${FORK_HEIGHT:-999999999}" ]]; then
      echo "✓ height=${H} (halt window before fork ${FORK_HEIGHT:-?})"
      exit 0
    fi
    if [[ -n "${FORK_HEIGHT:-}" && "${H}" -ge "${FORK_HEIGHT}" ]]; then
      echo "ERROR: height=${H} already reached/passed ForkHeight ${FORK_HEIGHT}; re-init"
      exit 1
    fi
    if [[ "${H}" -gt "${TARGET}" ]]; then
      echo "ERROR: height=${H} already passed target ${TARGET}; re-init with higher FORK_HEIGHT"
      exit 1
    fi
  fi
  echo "  height=${H:-?} (want >=${TARGET}, <${FORK_HEIGHT:-fork})"
  sleep 0.5
done
