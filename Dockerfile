# syntax=docker/dockerfile:1

# Please, when adding/editing this Dockerfile also take care of Dockerfile.cosmovisor as well
# Pinned to match go.mod's `toolchain go1.23.8` exactly — the golang:1.22-alpine
# base's GOTOOLCHAIN=local blocks the usual auto-download-newer-toolchain
# mechanism, so `go mod download` fails with "go.mod requires go >= 1.23.6"
# unless the base image already ships that version.
ARG GO_VERSION="1.23.8"
ARG RUNNER_IMAGE="alpine:3.18"
ARG BUILD_TAGS="netgo,ledger,muslc"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM golang:${GO_VERSION}-alpine3.20 as builder
ENV GO_PATH="/go"
ARG GIT_VERSION
ARG GIT_COMMIT
ARG BUILD_TAGS

RUN apk add --no-cache \
  ca-certificates \
  build-base \
  linux-headers \
  binutils-gold \
  xz \
  git \
  openssh-client

# Download go dependencies
WORKDIR /oraichain
## debug only
# COPY debug debug
COPY go.mod go.sum ./
# go.mod redirects github.com/CosmWasm/wasmvm/v2 to the private security-fix fork
# github.com/CosmWasm/priv_wasmvm_sec (embargoed, no public release for this version).
# Fetched over SSH using the builder's own collaborator access — forward the host's
# ssh-agent at build time:
#   docker buildx build --ssh default -f Dockerfile ...
# (classic/fine-grained PATs and OAuth tokens are rejected by the org for this repo;
# SSH is the only access path for an outside collaborator.)
ENV GOPRIVATE=github.com/CosmWasm/wasmvm,github.com/CosmWasm/priv_wasmvm_sec
ENV GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new"
# --mount=type=ssh only forwards the agent socket (so `ssh` can authenticate) — it
# does NOT rewrite HTTPS module URLs to SSH. go.mod's `replace` sends Go to
# https://github.com/CosmWasm/priv_wasmvm_sec; redirect that to SSH explicitly,
# same as the host-side git config (this container has no access to ~/.gitconfig).
RUN git config --global \
  url."ssh://git@github.com/CosmWasm/priv_wasmvm_sec".insteadOf \
  "https://github.com/CosmWasm/priv_wasmvm_sec"
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/root/go/pkg/mod \
  --mount=type=ssh \
  go mod download

# Cosmwasm - libwasmvm_muslc: the private fork ships the static lib compressed
# (.a.xz) inside the Go module. The linker looks for the plain .a via a -L flag
# that cgo hardcodes to the module's own internal/api/ dir (${SRCDIR}-relative,
# NOT /lib) — decompress it there in place (module cache files are read-only;
# chmod first). Integrity is covered by go.sum instead of a separate sha256 check.
RUN --mount=type=cache,target=/root/go/pkg/mod \
  set -eux; \
  ARCH="$(uname -m)"; \
  API_DIR="$(go list -m -f '{{.Dir}}' github.com/CosmWasm/wasmvm/v2)/internal/api"; \
  chmod u+w "${API_DIR}"; \
  unxz -k -f "${API_DIR}/libwasmvm_muslc.${ARCH}.a.xz"


# Copy the remaining files
COPY . .

# Build oraid binary (module cache + ssh forwarded again: this RUN doesn't inherit the
# filesystem state of the cache-mounted `go mod download` step above).
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/root/go/pkg/mod \
  --mount=type=ssh \
  LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make install
RUN echo "Ensuring binary is statically linked ..." \
  && (file /go/bin/oraid | grep -E "statically linked|static-pie linked")

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM ${RUNNER_IMAGE}

COPY --from=builder /go/bin/oraid /bin/oraid
ENV HOME=/oraichain
WORKDIR /oraichain

EXPOSE 26656
EXPOSE 26657
EXPOSE 1317
# Note: uncomment the line below if you need pprof in localoraichain
# We disable it by default in out main Dockerfile for security reasons
# EXPOSE 6060

ENTRYPOINT ["oraid"]