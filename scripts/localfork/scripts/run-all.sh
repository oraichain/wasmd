#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}/scripts"
chmod +x ./*.sh

export FORK_HEIGHT="${FORK_HEIGHT:-60}"
export OLD_TAG="${OLD_TAG:-v0.50.13b}"

# 1) old binary only — new image rebuilt after CW20 deploy (needs RecoveryAddress = tester)
./00-build.sh old
./01-init.sh
./02-start-old.sh
unset CW20_TO || true
./02b-deploy-cw20.sh

# shellcheck disable=SC1091
source "${ROOT_DIR}/data/cw20.env"
export CW20_TO
if [[ -z "${CW20_TO}" ]]; then
  echo "ERROR: CW20_TO / RecoveryAddress empty"
  exit 1
fi

./03-wait-height.sh

# Freeze consensus (stop S) while A stays up for balance queries, then halt A.
# Do NOT snapshot after A is stopped — LCD/RPC would be down.
cd "${ROOT_DIR}"
docker compose stop node-s
sleep 4
H=$(curl -sf http://127.0.0.1:26657/status | jq -r '.result.sync_info.latest_block_height')
echo "  frozen height=${H} after stop S"
cd "${ROOT_DIR}/scripts"
./snapshot-balances.sh pre
./04-halt.sh

# 2) rebuild new with fork height + RecoveryAddress ldflag (RecoveryAssets hardcoded)
./00-build.sh new
if ! docker image inspect localfork-new:local >/dev/null 2>&1; then
  echo "ERROR: localfork-new:local image missing after build — A/S cannot restart with fork binary"
  exit 1
fi
./05-restart-new.sh
./06-verify.sh

./snapshot-balances.sh post

echo "✓ localfork flow complete"
echo "  pre-fork balances:  ${ROOT_DIR}/data/balances-pre-fork.json"
echo "  post-fork balances: ${ROOT_DIR}/data/balances-post-fork.json"
