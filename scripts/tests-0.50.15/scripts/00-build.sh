#!/usr/bin/env bash
# Build the three images:
#   tests-0.50.15-seed:local <- git archive v0.50.13b (EVM era: mounts evm stores, resolves the Any)
#   tests-0.50.15-old:local  <- git archive v0.50.14  (EVM soft-removed via fork; stores still mounted)
#   tests-0.50.15-new:local  <- workspace             (make build => PIE + evmlegacy stub + v0.50.15
#                                                      upgrade that drops the evm stores)
#
#   MODE=seed|old|new|all  (default all)
set -euo pipefail
source "$(dirname "$0")/lib.sh"

MODE="${1:-all}"

build_from_tag() {
  local tag="$1" image="$2"
  git -C "${REPO_ROOT}" rev-parse -q --verify "refs/tags/${tag}" >/dev/null || die "git tag ${tag} not found"
  local tmp; tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp}"' RETURN
  log "building ${image} from tag ${tag}"
  git -C "${REPO_ROOT}" archive "${tag}" | tar -x -C "${tmp}"
  docker build -f "${ROOT_DIR}/Dockerfile.old" -t "${image}" "${tmp}"
  ok "${image}  (ELF Type: $(image_elf_type "${image}"))"
}

build_new() {
  log "building ${IMAGE_NEW} from workspace via 'make build'"
  docker build -f "${ROOT_DIR}/Dockerfile.new" -t "${IMAGE_NEW}" "${REPO_ROOT}"
  ok "${IMAGE_NEW}  (ELF Type: $(image_elf_type "${IMAGE_NEW}"))"
}

case "${MODE}" in
  seed) build_from_tag "${SEED_TAG}" "${IMAGE_SEED}" ;;
  old)  build_from_tag "${OLD_TAG}"  "${IMAGE_OLD}" ;;
  new)  build_new ;;
  all)  build_from_tag "${SEED_TAG}" "${IMAGE_SEED}"; build_from_tag "${OLD_TAG}" "${IMAGE_OLD}"; build_new ;;
  *)    die "usage: $0 [seed|old|new|all]" ;;
esac
