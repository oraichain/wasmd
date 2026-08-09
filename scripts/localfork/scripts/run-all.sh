#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}/scripts"
chmod +x ./*.sh

export FORK_HEIGHT="${FORK_HEIGHT:-20}"
export OLD_TAG="${OLD_TAG:-v0.50.13b}"

./00-build.sh
./01-init.sh
./02-start-old.sh
./03-wait-height.sh
./04-halt.sh
./05-restart-new.sh
./06-verify.sh

echo "✓ localfork flow complete"
