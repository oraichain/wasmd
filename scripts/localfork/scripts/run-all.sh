#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}/scripts"
chmod +x ./*.sh

export FORK_HEIGHT="${FORK_HEIGHT:-40}"
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
./snapshot-balances.sh pre

./04-halt.sh

# 2) rebuild new with fork height + RecoveryAddress ldflag (RecoveryAssets hardcoded)
./00-build.sh new
./05-restart-new.sh
./06-verify.sh

./snapshot-balances.sh post

echo "✓ localfork flow complete"
echo "  pre-fork balances:  ${ROOT_DIR}/data/balances-pre-fork.json"
echo "  post-fork balances: ${ROOT_DIR}/data/balances-post-fork.json"
