-- This query aggregates the block events into an array: 'events' exactly like the LCD 'events' format for simple handling
SELECT
	block_id,
	height,
	chain_id,
	jsonb_agg(
		json_build_object('type', type, 'attributes', attributes)
	) as events
FROM
	(
		SELECT
			block_id,
			height,
			chain_id,
			type,
			-- builds an array of objects with {'key': '<>', 'value':'<>'}
			json_agg(json_build_object('key', key, 'value', value)) AS attributes
		FROM
			blocks
			JOIN event_attributes ON (blocks.rowid = event_attributes.block_id)
		GROUP BY
			block_id,
			height,
			chain_id,
			type
	)
GROUP BY
	block_id,
	height,
	chain_id
order by
	height;