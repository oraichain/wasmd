#!/usr/bin/env bash
set -euo pipefail

RPC="${RPC:-http://127.0.0.1:26657}"
TARGET="${1:-}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ -z "${TARGET}" ]]; then
  if [[ -f "${ROOT_DIR}/data/meta.env" ]]; then
    # shellcheck disable=SC1091
    source "${ROOT_DIR}/data/meta.env"
    TARGET=$((FORK_HEIGHT - 2))
  else
    TARGET=18
  fi
fi

echo "==> Waiting until height == ${TARGET} (must halt BEFORE fork)"
while true; do
  H=$(curl -sf "${RPC}/status" | jq -r '.result.sync_info.latest_block_height // empty' || true)
  if [[ -n "${H}" ]]; then
    if [[ "${H}" -eq "${TARGET}" ]]; then
      echo "✓ height=${H}"
      exit 0
    fi
    if [[ "${H}" -gt "${TARGET}" ]]; then
      echo "ERROR: height=${H} already passed target ${TARGET}; re-init with higher FORK_HEIGHT"
      exit 1
    fi
  fi
  echo "  height=${H:-?} (want ${TARGET})"
  sleep 0.5
done
