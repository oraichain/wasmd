# syntax=docker/dockerfile:1
# Shared image build for localfork old/new binaries.
# - old: build context = git archive of tag v0.50.13b (no fork code)
# - new: build context = current workspace (fork enabled)

ARG GO_VERSION=1.23.8

FROM golang:${GO_VERSION}-bookworm AS builder

ARG ENABLE_FORK_LDFLAGS=false
ARG FORK_HEIGHT=20
ARG CW20_TO_ADDR=
ARG BUILD_TAGS=netgo
# When building localfork-new, append ",localfork" so constants_localfork.go is used.
ARG EXTRA_BUILD_TAGS=
ARG WASMVM_VERSION=v2.1.3

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates build-essential git curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Resolve wasmvm version from go.mod when possible; fallback to ARG.
RUN set -eux; \
    ARCH=$(uname -m); \
    case "$ARCH" in \
      x86_64) LIB=libwasmvm.x86_64.so ;; \
      aarch64|arm64) LIB=libwasmvm.aarch64.so ;; \
      *) echo "unsupported arch $ARCH"; exit 1 ;; \
    esac; \
    VER=$(go list -m -f '{{.Version}}' github.com/CosmWasm/wasmvm/v2 2>/dev/null || echo "${WASMVM_VERSION}"); \
    curl -fsSL -o "/usr/lib/${LIB}" \
      "https://github.com/CosmWasm/wasmvm/releases/download/${VER}/${LIB}"; \
    ln -sf "/usr/lib/${LIB}" /usr/lib/libwasmvm.so

ENV LD_LIBRARY_PATH=/usr/lib

RUN set -eux; \
    TAGS="${BUILD_TAGS}"; \
    if [ -n "${EXTRA_BUILD_TAGS}" ]; then \
      TAGS="${TAGS},${EXTRA_BUILD_TAGS}"; \
    fi; \
    LDFLAGS="-X github.com/cosmos/cosmos-sdk/version.Name=orai \
      -X github.com/cosmos/cosmos-sdk/version.AppName=oraid \
      -X github.com/CosmWasm/wasmd/app.Bech32Prefix=orai"; \
    if [ "${ENABLE_FORK_LDFLAGS}" = "true" ]; then \
      LDFLAGS="${LDFLAGS} -X github.com/CosmWasm/wasmd/app/upgrades/v05014.ForkHeightStr=${FORK_HEIGHT}"; \
      if [ -n "${CW20_TO_ADDR}" ]; then \
        LDFLAGS="${LDFLAGS} -X github.com/CosmWasm/wasmd/app/upgrades/v05014.RecoveryAddress=${CW20_TO_ADDR}"; \
      fi; \
    fi; \
    echo "building oraid tags=${TAGS}"; \
    go build -mod=readonly -tags "${TAGS}" -ldflags "${LDFLAGS}" \
      -o /out/oraid ./cmd/wasmd

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl jq \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/oraid /usr/bin/oraid
COPY --from=builder /usr/lib/libwasmvm*.so /usr/lib/
RUN ldconfig || true

ENV LD_LIBRARY_PATH=/usr/lib
ENV HOME=/orai
WORKDIR /orai

EXPOSE 26656 26657 1317 9090

ENTRYPOINT ["oraid"]
