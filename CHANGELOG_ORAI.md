## Unreleased

### BREAKING CHANGE

- ABCI 2.0, inheriting numerous new features like: IAVL enhancing
  performance; ABCI Optimistic Execution reducing block time,
  custom mempools allowing private transactions, front-running,...
  ([\#19](https://github.com/oraichain/wasmd/issues/19))
- Upgrade Oraichain mainnet to Cosmos SDK 0.50.10 with module
  enhancements like gov module emergency proposals; advanced IBC features
  ([\#19](https://github.com/oraichain/wasmd/issues/19))
- IAVL 1.0 enhances performance
  ([\#19](https://github.com/oraichain/wasmd/issues/19))

### BUG FIXES

- Support reading old cosmwasm proposals by adding backward compatible logic
  ([\#16](https://github.com/oraichain/wasmd/issues/16))
- Decrease inflation max to be equal to min when upgrading
  ([\#17](https://github.com/oraichain/wasmd/issues/17))

### IMPROVEMENTS

- Drastically reduce load time when running a forked node using a heavy genesis
  state. Also, significantly reduce memory consumption when loading a large
  genesis file ([\#20](https://github.com/oraichain/wasmd/issues/20))
