#!/usr/bin/env bash
# Full production-fixture fork e2e (local processes, no Docker).
set -euo pipefail
cd "$(dirname "$0")"
chmod +x ./*.sh

./00-build.sh all
./01-bootstrap-contracts.sh
./02-init.sh
./03-start-old.sh
./04-halt.sh
./05-restart-new.sh
./06-verify.sh

echo
echo "==> Scenario tests"
./07-chainid-guard.sh
./08-panic-scenario.sh

# Leave nothing running.
source ./env.sh; stop_all
echo
echo "✓ forkprod complete"
