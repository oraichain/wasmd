<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [cosmwasm/feemarket/v1/feemarket.proto](#cosmwasm/feemarket/v1/feemarket.proto)
    - [Params](#cosmwasm.feemarket.v1.Params)
  
- [cosmwasm/feemarket/v1/genesis.proto](#cosmwasm/feemarket/v1/genesis.proto)
    - [GenesisState](#cosmwasm.feemarket.v1.GenesisState)
  
- [cosmwasm/feemarket/v1/query.proto](#cosmwasm/feemarket/v1/query.proto)
    - [QueryBaseFeeRequest](#cosmwasm.feemarket.v1.QueryBaseFeeRequest)
    - [QueryBaseFeeResponse](#cosmwasm.feemarket.v1.QueryBaseFeeResponse)
    - [QueryBlockGasRequest](#cosmwasm.feemarket.v1.QueryBlockGasRequest)
    - [QueryBlockGasResponse](#cosmwasm.feemarket.v1.QueryBlockGasResponse)
    - [QueryParamsRequest](#cosmwasm.feemarket.v1.QueryParamsRequest)
    - [QueryParamsResponse](#cosmwasm.feemarket.v1.QueryParamsResponse)
  
    - [Query](#cosmwasm.feemarket.v1.Query)
  
- [Scalar Value Types](#scalar-value-types)



<a name="cosmwasm/feemarket/v1/feemarket.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/feemarket/v1/feemarket.proto



<a name="cosmwasm.feemarket.v1.Params"></a>

### Params
Params defines the EVM module parameters


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `no_base_fee` | [bool](#bool) |  | no base fee forces the EIP-1559 base fee to 0 (needed for 0 price calls) |
| `base_fee_change_denominator` | [uint32](#uint32) |  | base fee change denominator bounds the amount the base fee can change between blocks. |
| `elasticity_multiplier` | [uint32](#uint32) |  | elasticity multiplier bounds the maximum gas limit an EIP-1559 block may have. |
| `enable_height` | [int64](#int64) |  | height at which the base fee calculation is enabled. |
| `base_fee` | [string](#string) |  | base fee for EIP-1559 blocks. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="cosmwasm/feemarket/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/feemarket/v1/genesis.proto



<a name="cosmwasm.feemarket.v1.GenesisState"></a>

### GenesisState
GenesisState defines the feemarket module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#cosmwasm.feemarket.v1.Params) |  | params defines all the paramaters of the module. |
| `block_gas` | [uint64](#uint64) |  | block gas is the amount of gas used on the last block before the upgrade. Zero by default. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="cosmwasm/feemarket/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/feemarket/v1/query.proto



<a name="cosmwasm.feemarket.v1.QueryBaseFeeRequest"></a>

### QueryBaseFeeRequest
QueryBaseFeeRequest defines the request type for querying the EIP1559 base
fee.






<a name="cosmwasm.feemarket.v1.QueryBaseFeeResponse"></a>

### QueryBaseFeeResponse
BaseFeeResponse returns the EIP1559 base fee.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `base_fee` | [string](#string) |  |  |






<a name="cosmwasm.feemarket.v1.QueryBlockGasRequest"></a>

### QueryBlockGasRequest
QueryBlockGasRequest defines the request type for querying the EIP1559 base
fee.






<a name="cosmwasm.feemarket.v1.QueryBlockGasResponse"></a>

### QueryBlockGasResponse
QueryBlockGasResponse returns block gas used for a given height.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `gas` | [int64](#int64) |  |  |






<a name="cosmwasm.feemarket.v1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest defines the request type for querying x/evm parameters.






<a name="cosmwasm.feemarket.v1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse defines the response type for querying x/evm parameters.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#cosmwasm.feemarket.v1.Params) |  | params define the evm module parameters. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="cosmwasm.feemarket.v1.Query"></a>

### Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Params` | [QueryParamsRequest](#cosmwasm.feemarket.v1.QueryParamsRequest) | [QueryParamsResponse](#cosmwasm.feemarket.v1.QueryParamsResponse) | Params queries the parameters of x/feemarket module. | GET|/cosmwasm/feemarket/v1/params|
| `BaseFee` | [QueryBaseFeeRequest](#cosmwasm.feemarket.v1.QueryBaseFeeRequest) | [QueryBaseFeeResponse](#cosmwasm.feemarket.v1.QueryBaseFeeResponse) | BaseFee queries the base fee of the parent block of the current block. | GET|/cosmwasm/feemarket/v1/base_fee|
| `BlockGas` | [QueryBlockGasRequest](#cosmwasm.feemarket.v1.QueryBlockGasRequest) | [QueryBlockGasResponse](#cosmwasm.feemarket.v1.QueryBlockGasResponse) | BlockGas queries the gas used at a given block height | GET|/cosmwasm/feemarket/v1/block_gas|

 <!-- end services -->



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

