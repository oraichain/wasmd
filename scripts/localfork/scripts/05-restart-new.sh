#!/usr/bin/env bash
# Restart A and S with NEW binary; keep B on OLD binary.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

cat > docker-compose.override.yml <<EOF
services:
  node-a:
    image: localfork-new:local
  node-b:
    image: localfork-old:local
  node-s:
    image: localfork-new:local
EOF

echo "==> Starting node-a + node-s with localfork-new"
docker compose up -d --force-recreate node-a node-s

# Ensure B still on old image
docker compose up -d --force-recreate node-b

echo "✓ A+S=new, B=old"
docker compose ps
