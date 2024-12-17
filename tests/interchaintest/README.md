# Interchain Tests

## Prerequisites

- Docker - local environment for testing with different chains.
- Go - for running the tests

## Quick start

Build the local interchain test image:

```bash
make docker-build-debug
```

Then, try running an interchain test:

```bash
make ictest-basic
```