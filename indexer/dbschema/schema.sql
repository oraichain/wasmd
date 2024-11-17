/*
 This file defines the database schema for the PostgresQL ("psql") event sink
 implementation in CometBFT. The operator must create a database and install
 this schema before using the database to index events.
 */
-- The blocks table records metadata about each block.
-- The block record does not include its events or transactions (see tx_results).
CREATE TABLE blocks (
  rowid BIGSERIAL PRIMARY KEY,
  height BIGINT NOT NULL,
  chain_id VARCHAR NOT NULL,
  -- When this block header was logged into the sink, in UTC.
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (height, chain_id)
);

-- Index blocks by height and chain, since we need to resolve block IDs when
-- indexing transaction records and transaction events.
CREATE INDEX idx_blocks_height_chain ON blocks(height desc, chain_id);

-- The tx_results table records metadata about transaction results.  Note that
-- the events from a transaction are stored separately.
CREATE TABLE tx_results (
  rowid BIGSERIAL PRIMARY KEY,
  -- The block to which this transaction belongs.
  block_id BIGINT NOT NULL REFERENCES blocks(rowid),
  -- The sequential index of the transaction within the block.
  "index" INTEGER NOT NULL,
  height BIGINT NOT NULL,
  -- When this result record was logged into the sink, in UTC.
  created_at TIMESTAMPTZ NOT NULL,
  -- The hex-encoded hash of the transaction.
  tx_hash VARCHAR NOT NULL,
  -- The protobuf wire encoding of the TxResult message.
  tx_result BYTEA NOT NULL,
  UNIQUE (block_id, "index")
);

-- index based on hash for quick query
CREATE INDEX idx_tx_results_hash ON tx_results(tx_hash);

-- index based on height for quick query
CREATE INDEX idx_tx_results_height ON tx_results(height);

-- block id and index forms a unique transaction in both tables.
CREATE INDEX idx_tx_results_block_id_index ON tx_results(block_id desc, "index" desc);

-- The events table records events. All events (both block and transaction) are
-- associated with a block ID; transaction events also have a transaction ID.
CREATE TABLE events (
  rowid BIGSERIAL PRIMARY KEY,
  -- The block and transaction this event belongs to.
  -- If tx_id is NULL, this is a block event.
  block_id BIGINT NOT NULL REFERENCES blocks(rowid),
  tx_id BIGINT NULL REFERENCES tx_results(rowid),
  -- The application-defined type label for the event.
  type VARCHAR NOT NULL
);

-- The attributes table records event attributes.
CREATE TABLE attributes (
  event_id BIGINT NOT NULL REFERENCES events(rowid),
  key VARCHAR NOT NULL,
  -- bare key
  composite_key VARCHAR NOT NULL,
  -- composed type.key
  value VARCHAR NULL,
  UNIQUE (event_id, key)
);

-- Index attributes composite key & value so we can filter attributes easier
CREATE INDEX idx_attributes_composite_key_value ON attributes(composite_key, value);

-- index key-value pairs with value as numeric so when we do non-height range filter -> can enable indexing 
CREATE INDEX idx_attributes_value_cast ON attributes (composite_key, (CAST(value AS numeric)))
WHERE
  value ~ '^\d+$';

-- A joined view of events and their attributes. Events that do not have any
-- attributes are represented as a single row with empty key and value fields.
CREATE VIEW event_attributes AS
SELECT
  block_id,
  tx_id,
  type,
  key,
  composite_key,
  value
FROM
  events
  LEFT JOIN attributes ON (events.rowid = attributes.event_id);

-- A joined view of all block events (those having tx_id NULL).
CREATE VIEW block_events AS
SELECT
  blocks.rowid as block_id,
  height,
  chain_id,
  type,
  key,
  composite_key,
  value
FROM
  blocks
  JOIN event_attributes ON (blocks.rowid = event_attributes.block_id)
WHERE
  event_attributes.tx_id IS NULL;

-- A joined view of all transaction events.
CREATE VIEW tx_events AS
SELECT
  blocks.height,
  "index",
  chain_id,
  type,
  key,
  composite_key,
  value,
  tx_results.created_at
FROM
  blocks
  JOIN tx_results ON (blocks.rowid = tx_results.block_id)
  JOIN event_attributes ON (tx_results.rowid = event_attributes.tx_id)
WHERE
  event_attributes.tx_id IS NOT NULL;

-- with filtered_tx_event_attributes as (
--   SELECT
--     events.block_id,
--     height,
--     tx_id,
--     type,
--     key,
--     composite_key,
--     value
--   FROM
--     events
--     JOIN attributes ON (events.rowid = attributes.event_id)
--     join blocks on (events.block_id = blocks.rowid)
--   where
--     tx_id is NOT null -- and filter based on heights here
--     and height > 30
--     and height < 40
--   ORDER BY
--     tx_id desc
-- ),
-- filtered_tx_ids as (
--   select
--     tx_id
--   from
--     filtered_tx_event_attributes te2
--   where
--     te2.composite_key = 'message.module'
--     AND te2.value = 'bank'
-- )
-- select
--   tx.height,
--   tx.created_at,
--   tx.tx_hash,
--   messages,
--   memo,
--   fee,
--   tr.tx_result
-- from
--   filtered_tx_ids ftx
--   join tx_results tr on tr.rowid = ftx.tx_id
--   );