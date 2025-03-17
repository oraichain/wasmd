## Ante
When making a transaction, users are required to pay fees in the native token. However, the txfees module allows users to pay fees not only in the native token but also in other tokens. To enable this functionality, the fee payment process must be adjusted through two ante handlers: `MempoolFeeDecorator` and `DeducFeeDecorator`.

`MempoolFeeDecorator` is responsible for verifying whether the fee token is permitted for use in transaction payments. Additionally, it calculates the equivalent amount of the fee token in native tokens when users initiate a transaction, based on the time-weighted average price (TWAP) between the fee denomination and the native denomination on OraiDex.

`DeducFeeDecorator` ante handler will deduct the fee tokens into the appropriate fee pool (either FeeCollector or NonNativeFeeCollector). The accumulated fee tokens in the NonNativeFeeCollector will be swapped on OraiDex at the end of each epoch and subsequently transferred back to the FeeCollector.