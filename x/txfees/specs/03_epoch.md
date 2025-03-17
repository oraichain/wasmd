## Epoch

The `txfees` leverage the Osmosis `epoch` which is used to schedule module requests. These are `QueryTokenTwapExchangeRate` and `SwapNonNativeFeeToken`.

The `QueryTokenTwapExchangeRate` is used to query Time-Weighted Average Price (TWAP) data from OraiDex.

The `SwapNonNativeFeeToken` is used to swap accumulated non-native fees token in Non-Native Fee Collector pool for `ORAI` on OraiDex.