# Indexer

This package is responsible for:

- Connecting with a Postgres DB for block and txs indexing
- Allowing the [ABCI streaming plugin](../streaming/README.md) to insert custom module events into the DB

## Prerequisites

To start indexing blocks and txs to Postgres locally, you need:

- Postgres installed. It's best to use Docker to start the DBMS -> Install Docker
- Docker-compose. It's convenient to use docker-compose to quickly start the Postgres docker container.

## Quick start

### Start the DBMS

Follow the below steps to start the indexer:

```bash
cd indexer/

# Start postgres
docker-compose up -d
```

### Configuration

You need to configure the `config.toml` file as follows to enable the indexer via Postgres:

```toml
#######################################################
###   Transaction Indexer Configuration Options     ###
#######################################################
[tx_index]

# What indexer to use for transactions
#
# The application will set which txs to index. In some cases a node operator will be able
# to decide which txs to index based on configuration set in the application.
#
# Options:
#   1) "null"
#   2) "kv" (default) - the simplest possible indexer, backed by key-value storage (defaults to levelDB; see DBBackend).
# 		- When "kv" is chosen "tx.height" and "tx.hash" will always be indexed.
#   3) "psql" - the indexer services backed by PostgreSQL.
# When "kv" or "psql" is chosen "tx.height" and "tx.hash" will always be indexed.
indexer = "psql"

# The PostgreSQL connection configuration, the connection format:
#   postgresql://<user>:<password>@<host>:<port>/<db>?<opts>

# sslmode=disable for local
psql-conn = "postgresql://admin:root@localhost:5432/node_indexer?sslmode=disable"
```

## Interacting with the DBMS

There are several tools that can do the trick: **pgAdmin**, **dBeaver**, ...