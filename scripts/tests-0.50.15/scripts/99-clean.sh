#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"
compose down -v --remove-orphans 2>/dev/null || true
rm -rf "${DATA_DIR}" "${ROOT_DIR}/docker-compose.override.yml"
ok "cleaned data/ + containers (images kept; 'docker rmi ${IMAGE_OLD} ${IMAGE_NEW}' to drop them)"
