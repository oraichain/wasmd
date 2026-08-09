#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"
FORK_HEIGHT="${FORK_HEIGHT:-20}"
OLD_TAG="${OLD_TAG:-v0.50.13b}"
DOCKERFILE="${ROOT_DIR}/Dockerfile"

cd "${REPO_ROOT}"

if ! git rev-parse -q --verify "refs/tags/${OLD_TAG}" >/dev/null; then
  echo "ERROR: git tag ${OLD_TAG} not found"
  exit 1
fi

TMP_OLD="$(mktemp -d)"
cleanup() { rm -rf "${TMP_OLD}"; }
trap cleanup EXIT

echo "==> Building localfork-old from tag ${OLD_TAG}"
git archive "${OLD_TAG}" | tar -x -C "${TMP_OLD}"
# Use the Dockerfile from current workspace (tag may not have scripts/localfork)
docker build \
  -f "${DOCKERFILE}" \
  --build-arg ENABLE_FORK_LDFLAGS=false \
  -t localfork-old:local \
  "${TMP_OLD}"

echo "==> Building localfork-new from current workspace (fork ON, FORK_HEIGHT=${FORK_HEIGHT})"
docker build \
  -f "${DOCKERFILE}" \
  --build-arg ENABLE_FORK_LDFLAGS=true \
  --build-arg FORK_HEIGHT="${FORK_HEIGHT}" \
  -t localfork-new:local \
  "${REPO_ROOT}"

echo "✓ Images ready:"
echo "  localfork-old:local  <= git tag ${OLD_TAG}"
echo "  localfork-new:local  <= HEAD + fork (height=${FORK_HEIGHT})"
