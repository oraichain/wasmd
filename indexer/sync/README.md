# Sync block for indexer

This package is responsible for:

- Connecting with a Postgres DB for block and txs indexing
- Allowing sync data from specific block to indexer database

## Prerequisites

To start syncing blocks data to Postgres database, you need:

- An archive node, which stores all data of the chain and not pruning
- PostgreSql database which store indexed data of the chain

### Configuration

### Configuration

You need to configure the `config.toml` file located at `/<your-path-to-node-dir>/.oraid/config/config.toml` as follows to synchronize blocks into the indexer via Postgres:

```toml
#######################################################
###   Transaction Indexer Configuration Options     ###
#######################################################
[tx_index]

indexer = "null"

# The PostgreSQL connection configuration, the connection format:
#   postgresql://<user>:<password>@<host>:<port>/<db>?<opts>

# sslmode=disable for local
psql-conn = "postgresql://admin:root@localhost:5432/node_indexer?sslmode=disable"
```

Also, make sure there exists a `client.toml` file located at `/<your-path-to-node-dir>/.oraid/config/client.toml` with a valid `chain-id` field.

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
