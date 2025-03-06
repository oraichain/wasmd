## State

The `txfees` module keeps state of two primary objects such as allowed token list and token exchange rate

### AllowedToken
Token that are permitted for use in paying transaction fees.

`0x00 | byte{token_denom} -> byte{1}`

### TokenExchangeRate
Exchange rate of specific token denom with `ORAI`

`0x01 | byte{token_denom} -> sdk.Dec`

It is important to note that the exchange rate will fluctuate over time, as it is determined by the time-weighted average price (TWAP) of the token on OraiDex. This implies that the exchange rate will reflect the average price of the token over a specified time period, rather than its instantaneous price.

### TokenPoolRoute

Pool Id of specific token with `ORAI` on OraiDex

`0x02 | byte{token_denom} -> index`
