# Run ICA migration tests

## Setup

Install Go relayer

```bash
#go relayer (make sure to use v2.0.0-rc4 or later!)
git clone https://github.com/cosmos/relayer.git
cd relayer && git checkout v2.5.2
make install
```

## Run the tests

```bash
./scripts/ica-migration-tests/e2e-ica-tests.sh
```