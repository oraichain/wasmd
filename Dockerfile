# syntax=docker/dockerfile:1

# Please, when adding/editing this Dockerfile also take care of Dockerfile.cosmovisor as well
ARG GO_VERSION="1.22"
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
  xz

# Download go dependencies
WORKDIR /oraichain
## debug only
# COPY debug debug
COPY go.mod go.sum ./
# github.com/CosmWasm/wasmvm/v2 is pinned to a private security-fix version (no public
# tag/release exists for it) — needs GOPRIVATE + git credentials for the private fork,
# supplied at build time via --secret id=gitconfig,src=<path to a gitconfig with those creds>.
ENV GOPRIVATE=github.com/CosmWasm/wasmvm
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/root/go/pkg/mod \
  --mount=type=secret,id=gitconfig,target=/root/.gitconfig \
  go mod download

# Cosmwasm - libwasmvm_muslc: the private fork ships the static lib as .xz inside the
# Go module itself (no public GitHub release asset for this version to download+pin).
# Integrity is covered by go.sum instead of a separate sha256 check.
RUN --mount=type=cache,target=/root/go/pkg/mod \
  ARCH="$(uname -m)" \
  && unxz -c "$(go list -m -f '{{.Dir}}' github.com/CosmWasm/wasmvm/v2)/internal/api/libwasmvm_muslc.$ARCH.a.xz" > "/lib/libwasmvm_muslc.$ARCH.a"


# Copy the remaining files
COPY . .

# Build oraid binary
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make install
RUN echo "Ensuring binary is statically linked ..." \
  && (file /go/bin/oraid | grep "statically linked")

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