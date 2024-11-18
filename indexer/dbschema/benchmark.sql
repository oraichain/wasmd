-- old slow query
WITH filtered_heights AS (
  SELECT
    distinct tx_results.rowid,
    height
  FROM
    tx_results
  ORDER BY
    height desc
),
filtered_tx_event_attributes as (
  SELECT
    events.block_id,
    height,
    tx_id,
    composite_key,
    value
  FROM
    events
    join filtered_heights fh on (fh.rowid = events.tx_id)
    JOIN attributes ON (events.rowid = attributes.event_id)
  order by
    events.block_id desc
),
filtered_tx_ids as (
  select
    distinct tx_id
  from
    filtered_tx_event_attributes te2
  where
    te2.composite_key = 'wasm-matched_order._contract_address'
    and te2.value = 'orai1nt58gcu4e63v7k55phnr3gaym9tvk3q4apqzqccjuwppgjuyjy6sxk8yzp'
  intersect
  select
    distinct tx_id
  from
    filtered_tx_event_attributes te3
  where
    te3.composite_key = 'wasm-matched_order.direction'
    and te3.value = 'Buy'
  intersect
  select
    distinct tx_id
  from
    filtered_tx_event_attributes te4
  where
    te4.composite_key = 'wasm-matched_order.order_id'
    and te4.value :: numeric > 4734390
)
select
  tr.height,
  tr.created_at,
  tr.tx_hash,
  tr.tx_result
from
  filtered_tx_ids ftx
  join tx_results tr on tr.rowid = ftx.tx_id explain analyze WITH filtered_heights AS (
    SELECT
      DISTINCT tx_results.rowid,
      height
    FROM
      tx_results
    ORDER BY
      height DESC
  ),
  filtered_tx_event_attributes AS (
    SELECT
      DISTINCT tx_id
    FROM
      events
      JOIN filtered_heights fh ON (fh.rowid = events.tx_id)
      JOIN attributes ON (events.rowid = attributes.event_id)
    WHERE
      composite_key = 'wasm-matched_order._contract_address'
      AND value = 'orai1nt58gcu4e63v7k55phnr3gaym9tvk3q4apqzqccjuwppgjuyjy6sxk8yzp'
    INTERSECT
    SELECT
      DISTINCT tx_id
    FROM
      events
      JOIN filtered_heights fh ON (fh.rowid = events.tx_id)
      JOIN attributes ON (events.rowid = attributes.event_id)
    WHERE
      composite_key = 'wasm-matched_order.direction'
      AND value = 'Buy'
  )
SELECT
  tr.height,
  tr.created_at,
  tr.tx_hash,
  tr.tx_result
FROM
  filtered_tx_event_attributes ftx
  JOIN tx_results tr ON tr.rowid = ftx.tx_id;


-- new optimized query
-- WITH filtered_heights AS (
--   SELECT
--     DISTINCT tx_results.rowid,
--     height
--   FROM
--     tx_results
--   ORDER BY
--     height DESC
-- ),
-- filtered_tx_event_attributes AS (
--   SELECT
--     DISTINCT tx_id
--   FROM
--     events
--     JOIN filtered_heights fh ON (fh.rowid = events.tx_id)
--     JOIN attributes ON (events.rowid = attributes.event_id)
--   WHERE
--     composite_key = 'wasm-matched_order._contract_address'
--     AND value = 'orai1nt58gcu4e63v7k55phnr3gaym9tvk3q4apqzqccjuwppgjuyjy6sxk8yzp'
--   INTERSECT
--   SELECT
--     DISTINCT tx_id
--   FROM
--     events
--     JOIN filtered_heights fh ON (fh.rowid = events.tx_id)
--     JOIN attributes ON (events.rowid = attributes.event_id)
--   WHERE
--     composite_key = 'wasm-matched_order.direction'
--     AND value = 'Buy'
-- )
-- SELECT
--   tr.height,
--   tr.created_at,
--   tr.tx_hash,
--   tr.tx_result
-- FROM
--   filtered_tx_event_attributes ftx
--   JOIN tx_results tr ON tr.rowid = ftx.tx_id;