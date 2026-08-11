#!/usr/bin/env bash
# Restart A+S on the NEW production binary; B deliberately stays on v0.50.13b.
set -euo pipefail
source "$(dirname "$0")/env.sh"
start_node a "${BIN_NEW}"; echo "  node-a -> oraid-prod (new)"
start_node s "${BIN_NEW}"; echo "  node-s -> oraid-prod (new)"
start_node b "${BIN_OLD}"; echo "  node-b -> oraid-old  (STALE, must diverge)"
sleep 8
echo "✓ A+S new, B old"
