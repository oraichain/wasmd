#!/usr/bin/env bash
# Build localfork Docker images.
#   MODE=old|new|all (default all)
# For MODE=new, optional CW20_TO (RecoveryAddress) is baked into ldflags after deploy.
# RecoveryAssets are hardcoded Instantiate2 addresses — no contract ldflag.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"
FORK_HEIGHT="${FORK_HEIGHT:-20}"
OLD_TAG="${OLD_TAG:-v0.50.13b}"
DOCKERFILE="${ROOT_DIR}/Dockerfile"
MODE="${1:-${BUILD_MODE:-all}}"

cd "${REPO_ROOT}"

if ! git rev-parse -q --verify "refs/tags/${OLD_TAG}" >/dev/null; then
  echo "ERROR: git tag ${OLD_TAG} not found"
  exit 1
fi

build_old() {
  local tmp_old
  tmp_old="$(mktemp -d)"
  echo "==> Building localfork-old from tag ${OLD_TAG}"
  git archive "${OLD_TAG}" | tar -x -C "${tmp_old}"
  docker build \
    -f "${DOCKERFILE}" \
    --build-arg ENABLE_FORK_LDFLAGS=false \
    -t localfork-old:local \
    "${tmp_old}"
  rm -rf "${tmp_old}"
}

build_new() {
  echo "==> Building localfork-new from current workspace (fork ON, FORK_HEIGHT=${FORK_HEIGHT}, tag localfork)"
  local args=(
    -f "${DOCKERFILE}"
    --build-arg ENABLE_FORK_LDFLAGS=true
    --build-arg FORK_HEIGHT="${FORK_HEIGHT}"
    --build-arg EXTRA_BUILD_TAGS=localfork
    -t localfork-new:local
  )
  if [[ -n "${CW20_TO:-}" ]]; then
    args+=(
      --build-arg "CW20_TO_ADDR=${CW20_TO}"
    )
    echo "  recovery ldflags: RecoveryAddress=${CW20_TO}"
    echo "  fixtures: constants_localfork.go (Instantiate2 CW20 + native denoms + scaled revert)"
  fi
  docker build "${args[@]}" "${REPO_ROOT}"
}

case "${MODE}" in
  old) build_old ;;
  new) build_new ;;
  all)
    build_old
    build_new
    ;;
  *)
    echo "USAGE: $0 [old|new|all]"
    exit 1
    ;;
esac

echo "✓ Image(s) ready (mode=${MODE}, fork_height=${FORK_HEIGHT})"
