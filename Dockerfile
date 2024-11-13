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
  binutils-gold

# Download go dependencies
WORKDIR /oraichain
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/root/go/pkg/mod \
  go mod download

# Cosmwasm - Download correct libwasmvm version
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.3/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.3/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep faea4e15390e046d2ca8441c21a88dba56f9a0363f92c5d94015df0ac6da1f2d
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep 8dab08434a5fe57a6fbbcb8041794bc3c31846d31f8ff5fb353ee74e0fcd3093


# Copy the remaining files
COPY . .

# Build oraid binary
RUN make build

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM ${RUNNER_IMAGE}

COPY --from=builder /go/bin/oraid /bin/oraid

WORKDIR /oraichain

EXPOSE 26656
EXPOSE 26657
EXPOSE 1317
# Note: uncomment the line below if you need pprof in localoraichain
# We disable it by default in out main Dockerfile for security reasons
# EXPOSE 6060

ENTRYPOINT ["oraid"]