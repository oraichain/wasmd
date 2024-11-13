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
docker-compose -f indexer/docker-compose.yml up -d
```

### Interacting with the DBMS

There are several tools that can do the trick: **pgAdmin**, **dBeaver**, ...

Use [these migration SQLs](./dbschema/schema.sql) to create tables for the indexer via **dBeaver** or your favorite migration script.

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

### Testing

Run the following script to start the indexer locally: `./scripts/setup_oraid_psql_indexer.sh`

### Search for txs

Example:

```bash
# query: tx.height >= 2760 AND tx.height <= 2761
curl "http://localhost:5050/tx_search?query=\"tx.height%20%3E%3D%202760%20AND%20tx.height%20%3C%3D%202761\""
# browser
http://localhost:5050/tx_search?query=%22tx.height%20%3E%3D%202760%20AND%20tx.height%20%3C%3D%202761%22

# query by tx hash:
curl "http://localhost:5050/tx_search?hash=\"E53292A4CAE48C347CDE93CF2FDD5C8511A56889EC6B0B0DD5221FF13ED2D5C8\""
# browser
http://localhost:5050/tx_search?hash=%22E53292A4CAE48C347CDE93CF2FDD5C8511A56889EC6B0B0DD5221FF13ED2D5C8%22

# filter by instantiate code id (browser)
http://localhost:5050/tx_search?query=%22instantiate.code_id%20>2%22
# >= 2
http://localhost:5050/tx_search?query=%22instantiate.code_id%20%3E=2%22
# >= 1 & < 3
http://localhost:5050/tx_search?query=%22instantiate.code_id%20%3E=1%20AND%20instantiate.code_id%3C3%22
```

Note that the query must use `AND` uppercase letter, not `and`

### Integration with Cosmjs

The Indexer RPC works out-of-the-box with Cosmjs. You only need to replace the RPC endpoint with the Indexer RPC endpoint, and it should work:

```ts
import { GasPrice, SigningStargateClient } from "@cosmjs/stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import dotenv from "dotenv";
import { ORAI } from "@oraichain/common";
dotenv.config();

(async () => {
  const signer = await DirectSecp256k1HdWallet.fromMnemonic(
    process.env.EXAMPLES_MNEMONIC,
    { prefix: ORAI }
  ); // replace your mnemonic here
  const accounts = await signer.getAccounts();
  const address = accounts[0].address;
  const client = await SigningStargateClient.connectWithSigner(
    "http://localhost:5050", // -> new Indexer RPC
    // "https://rpc.orai.io" -> original CometBFT RPC
    signer,
    { gasPrice: GasPrice.fromString("0.001orai") }
  );
})();

```
