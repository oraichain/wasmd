#!/usr/bin/env bash
# Build the three images:
#   tests-0.50.15-seed:local <- git archive v0.50.13b (EVM era: mounts evm stores, resolves the Any)
#   tests-0.50.15-old:local  <- git archive v0.50.14  (EVM soft-removed via fork; stores still mounted)
#   tests-0.50.15-new:local  <- workspace             (current checkout => treated as "v0.50.15":
#                                                      evmlegacy stub + v05015 store-prune upgrade
#                                                      + whatever security patch is on the branch)
#
# "old" and "new" are compiled for linux/amd64 via `docker buildx build --platform
# linux/amd64` and the resulting binary is extracted to bin/<name>/ on the host;
# the actual test images are then a trivial COPY-only package step (Dockerfile.
# {old,new}-runtime) with no Go toolchain and no SSH involved. This keeps the one
# step that needs the private-module SSH access (compiling "new") isolated to a
# single command, and every later rebuild of the runtime image is instant and
# credential-free.
#
#   MODE=seed|old|new|all  (default all)
set -euo pipefail
source "$(dirname "$0")/lib.sh"

MODE="${1:-all}"
PLATFORM="linux/amd64"

build_seed() {
  git -C "${REPO_ROOT}" rev-parse -q --verify "refs/tags/${SEED_TAG}" >/dev/null || die "git tag ${SEED_TAG} not found"
  local tmp; tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp}"' RETURN
  log "building ${IMAGE_SEED} from tag ${SEED_TAG} (native arch, bootstrap only)"
  git -C "${REPO_ROOT}" archive "${SEED_TAG}" | tar -x -C "${tmp}"
  docker build -f "${ROOT_DIR}/Dockerfile.old" -t "${IMAGE_SEED}" "${tmp}"
  ok "${IMAGE_SEED}  (ELF Type: $(image_elf_type "${IMAGE_SEED}"))"
}

# Compile a Dockerfile for linux/amd64 and extract /usr/bin/oraid to bin/<name>/
# on the host. If the binary is dynamically linked against libwasmvm, also
# extract the exact .so it needs (per its own DT_NEEDED entry — the binary links
# the arch-specific filename directly, e.g. libwasmvm.x86_64.so, not a generic
# libwasmvm.so symlink); a static-PIE binary has no such dependency and this is
# skipped. Extra buildx args (e.g. --ssh default) go after the context arg.
extract_amd64_binary() {
  local name="$1" dockerfile="$2" context="$3"; shift 3
  local builder_image="tests-0.50.15-${name}-amd64-builder:local"
  local out="${ROOT_DIR}/bin/${name}"

  log "compiling ${name} for ${PLATFORM} (docker buildx, $* )"
  docker buildx build --platform "${PLATFORM}" --load "$@" \
    -f "${dockerfile}" -t "${builder_image}" "${context}"

  rm -rf "${out}"; mkdir -p "${out}/lib"
  local cid; cid=$(docker create --platform "${PLATFORM}" "${builder_image}")
  docker cp "${cid}:/usr/bin/oraid" "${out}/oraid"

  local needed; needed=$(docker run --rm --platform "${PLATFORM}" --entrypoint readelf "${builder_image}" \
      -d /usr/bin/oraid 2>/dev/null | awk -F'[][]' '/NEEDED/ && /libwasmvm/ {print $2}')
  if [[ -n "${needed}" ]]; then
    docker cp "${cid}:/usr/lib/${needed}" "${out}/lib/${needed}"
    ok "extracted ${out}/oraid + lib/${needed} (dynamic)"
  else
    ok "extracted ${out}/oraid (static, no libwasmvm.so dependency)"
  fi

  docker rm "${cid}" >/dev/null
}

package_runtime() {
  local name="$1" image="$2" dockerfile="$3"
  [[ -x "${ROOT_DIR}/bin/${name}/oraid" ]] || die "no extracted binary at bin/${name}/oraid — run extract first"
  docker build --platform "${PLATFORM}" -f "${dockerfile}" -t "${image}" "${ROOT_DIR}"
  ok "${image}  (ELF Type: $(image_elf_type "${image}"))"
}

build_old() {
  local tmp; tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp}"' RETURN
  git -C "${REPO_ROOT}" rev-parse -q --verify "refs/tags/${OLD_TAG}" >/dev/null || die "git tag ${OLD_TAG} not found"
  git -C "${REPO_ROOT}" archive "${OLD_TAG}" | tar -x -C "${tmp}"
  extract_amd64_binary old "${ROOT_DIR}/Dockerfile.old" "${tmp}"
  package_runtime old "${IMAGE_OLD}" "${ROOT_DIR}/Dockerfile.old-runtime"
}

build_new() {
  log "new = current checkout: $(git -C "${REPO_ROOT}" branch --show-current) (not a git tag)"
  # --ssh default forwards the host's ssh-agent so `go mod download` can fetch the
  # private wasmvm security fork (go.mod replace) over SSH. Run this script inside
  # `ssh-agent bash -c '... ssh-add ...; ./00-build.sh new'` if your key isn't
  # already loaded in a running agent.
  extract_amd64_binary new "${ROOT_DIR}/Dockerfile.new" "${REPO_ROOT}" --ssh default
  package_runtime new "${IMAGE_NEW}" "${ROOT_DIR}/Dockerfile.new-runtime"
}

case "${MODE}" in
  seed) build_seed ;;
  old)  build_old ;;
  new)  build_new ;;
  all)  build_seed; build_old; build_new ;;
  *)    die "usage: $0 [seed|old|new|all]" ;;
esac
