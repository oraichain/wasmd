#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
chmod +x ./*.sh

./00-build.sh "${1:-all}"
./01-init.sh
./02-seed-run.sh         # run 3 nodes LIVE on v0.50.13b (EVM era); proposal 316 queryable
./03-pre-upgrade.sh      # hot swap v0.50.13b -> v0.50.14 (no proposal); app_hash continuity; 316 now broken
./04-gov-upgrade.sh      # gov software-upgrade -> v0.50.15 -> 316 readable, evm stores deleted; nodes left running
./05-restart-check.sh    # cold-restart on v0.50.15 — catches a missing app.go unmount for deleted stores
