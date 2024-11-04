INSERT INTO
  blocks (height, chain_id, created_at)
VALUES
  (
    10,
    -- height
    'Oraichain',
    -- chain_id
    NOW() -- created_at
  );

INSERT INTO
  tx_requests (
    block_id,
    index,
    created_at,
    tx_hash,
    messages,
    fee,
    memo,
    signer_pub_key,
    signer_pub_key_type,
    signer_mode,
    signer_sequence
  )
VALUES
  (
    10,
    -- block_id
    0,
    -- index
    NOW(),
    -- created_at
    '0xabc123...',
    -- tx_hash
    '\x4d65737361676544617461',
    -- messages (hex-encoded BYTEA)
    ROW(
      -- fee (tx_fee type)
      50000,
      -- gas_limit
      ARRAY [                                -- amount array
      ROW('1000', 'orai')::fee_amount,
      ROW('500', 'atom')::fee_amount
    ],
      'payer_address',
      -- payer
      'granter_address' -- granter
    ) :: tx_fee,
    'Transaction memo text',
    -- memo
    '\x030201',
    -- signer_pub_key (hex-encoded BYTEA)
    'ed25519',
    -- signer_pub_key_type
    'signing_mode',
    -- signer_mode
    12345 -- signer_sequence
  );

