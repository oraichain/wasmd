# Sync block for indexer

This package is responsible for:

- Connecting with a Postgres DB for block and txs indexing
- Allowing sync data from specific block to indexer database

## Prerequisites

To start syncing blocks data to Postgres database, you need:

- An archive node, which stores all data of the chain and not pruning
- PostgreSql database which store indexed data of the chain

## Quick start
To sync data from specific blocks, please following these steps below:

- Go to `sync` directory:
```sh
cd <path_to_WASMD_folder>/indexer/sync
```
- Export your home node directory (if not we will use default home node directory $HOME/.oraid)
```sh
export HOME_PATH=<path_to_home_node>
# example: export HOME_PATH=$PWD/.oraid
```
- Run follwing cmd:
```sh
go run main.go --archive-node <archive_node_rpc> --start-block <start_sync_block> --end-block <end_sync_block>
```
