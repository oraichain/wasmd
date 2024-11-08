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
CREATE INDEX idx_blocks_height_chain ON blocks(height, chain_id);

-- The tx_requests table records metadata about transaction requests.
CREATE TABLE tx_requests (
  rowid BIGSERIAL PRIMARY KEY,
  -- The block to which this transaction belongs.
  block_id BIGINT NOT NULL REFERENCES blocks(rowid),
  -- The sequential index of the transaction within the block.
  index INTEGER NOT NULL,
  -- block height
  height BIGINT NOT NULL,
  -- When this result record was logged into the sink, in UTC.
  created_at TIMESTAMPTZ NOT NULL,
  -- The hex-encoded hash of the transaction.
  tx_hash VARCHAR NOT NULL,
  -- messages of the transaction
  messages BYTEA NOT NULL,
  -- transaction fees
  fee VARCHAR NOT NULL,
  -- memo of the transaction
  memo VARCHAR,
  UNIQUE (block_id, index)
);

-- The tx_results table records metadata about transaction results.  Note that
-- the events from a transaction are stored separately.
CREATE TABLE tx_results (
  rowid BIGSERIAL PRIMARY KEY,
  -- The block to which this transaction belongs.
  block_id BIGINT NOT NULL REFERENCES blocks(rowid),
  -- The sequential index of the transaction within the block.
  index INTEGER NOT NULL,
  -- When this result record was logged into the sink, in UTC.
  created_at TIMESTAMPTZ NOT NULL,
  -- The hex-encoded hash of the transaction.
  tx_hash VARCHAR NOT NULL,
  -- The protobuf wire encoding of the TxResult message.
  tx_result BYTEA NOT NULL,
  -- code of the tx verifying if it's successful or not
  code INTEGER NOT NULL,
  -- extra useful data
  logs VARCHAR NOT NULL,
  info VARCHAR NOT NULL,
  gas_wanted BIGINT NOT NULL,
  gas_used BIGINT NOT NULL,
  codespace VARCHAR NOT NULL,
  UNIQUE (block_id, index)
);

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
  height,
  index,
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

CREATE VIEW json_attribute_events AS
SELECT
  tx_requests.block_id,
  height,
  tx_hash,
  type,
  jsonb_agg(
    jsonb_build_object(
      'key',
      key,
      'value',
      value
    )
  ) AS attributes
FROM
  tx_requests
  JOIN event_attributes ON (tx_requests.block_id = event_attributes.block_id)
WHERE
  event_attributes.tx_id IS NOT NULL
GROUP BY
  tx_requests.block_id,
  height,
  height,
  tx_hash,
  type;

-- select
--   ftx.height,
--   created_at,
--   ftx.tx_hash,
--   messages,
--   memo,
--   fee,
--   jsonb_agg(
--     DISTINCT jsonb_build_object('type', type, 'attributes', attributes)
--   ) as events
-- from
--   tx_requests ftx
--   join json_attribute_events jae on (ftx.height = jae.height)
--   join (
--     SELECT
--       t1.height
--     FROM
--       tx_events t1
--       JOIN tx_events t2 ON t1.height = t2.height
--     WHERE
--       (
--         t2.composite_key LIKE 'message.module%'
--         AND t2.value = 'bank'
--       )
--   ) as filter_table on filter_table.height = ftx.height
-- where
--   ftx.height > 30
--   and ftx.height < 40
-- group by
--   ftx.height,
--   created_at,
--   ftx.tx_hash,
--   messages,
--   memo,
--   fee;