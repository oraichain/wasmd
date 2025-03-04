## Proposals

The `txfees` module contains the following proposals:

### ProposalAddFeeToken
```go
type MsgAddFeeToken struct {
    Authority string
    TokenDenom string
    OraiDexTokenPoolId uint64
}
```
The `ProposalAddFeeToken` allows for the addition of new tokens as fee tokens.

### ProposalRemoveFeeToken
```go
type MsgRemoveFeeToken struct {
    Authority string
    TokenDenom string
}
```
The `ProposalRemoveFeeToken` facilitates the removal of a token from the list of fee tokens.

### UpdateParams
```go
type MsgUpdateParams struct {
    Authority string
    Params Params
}
```