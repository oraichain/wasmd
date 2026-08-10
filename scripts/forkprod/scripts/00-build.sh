#!/usr/bin/env bash
# Build both binaries locally (no Docker).
#   oraid-prod : current workspace, PRODUCTION fixtures (no localfork tag, no ldflags)
#   oraid-old  : v0.50.13b, for the stale validator
set -euo pipefail
source "$(dirname "$0")/env.sh"
OLD_TAG="${OLD_TAG:-v0.50.13b}"
MODE="${1:-all}"
mkdir -p "${DATA_DIR}"

build_new() {
  echo "==> Building oraid-prod (production fixtures, no localfork tag)"
  (cd "${REPO_ROOT}" && go build -o "${BIN_NEW}" ./cmd/wasmd)
}

build_old() {
  if ! (cd "${REPO_ROOT}" && git rev-parse -q --verify "refs/tags/${OLD_TAG}" >/dev/null); then
    echo "ERROR: git tag ${OLD_TAG} not found"; exit 1
  fi
  if [[ -x "${BIN_OLD}" ]]; then
    echo "==> oraid-old already built (delete ${BIN_OLD} to force)"; return
  fi
  local tmp; tmp="$(mktemp -d)"
  echo "==> Building oraid-old from ${OLD_TAG} (this takes a few minutes)"
  (cd "${REPO_ROOT}" && git archive "${OLD_TAG}" | tar -x -C "${tmp}")
  (cd "${tmp}" && go build -o "${BIN_OLD}" ./cmd/wasmd)
  rm -rf "${tmp}"
}

case "${MODE}" in
  new) build_new ;;
  old) build_old ;;
  all) build_new; build_old ;;
  *) echo "USAGE: $0 [new|old|all]"; exit 1 ;;
esac
echo "✓ binaries ready"
