#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

# Ensure all start on old binary
export COMPOSE_FILE=docker-compose.yml

# Force image to old for all services via compose override
cat > docker-compose.override.yml <<EOF
services:
  node-a:
    image: localfork-old:local
  node-b:
    image: localfork-old:local
  node-s:
    image: localfork-old:local
EOF

docker compose up -d
echo "✓ Started node-a, node-b, node-s on localfork-old"
docker compose ps
