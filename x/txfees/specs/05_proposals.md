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
The ProposalAddFeeToken allows for the addition of new tokens as fee tokens.

### ProposalRemoveFeeToken
```go
type MsgRemoveFeeToken struct {
    Authority string
    TokenDenom string
}
```
The ProposalRemoveFeeToken facilitates the removal of a token from the list of fee tokens.

### UpdateParams
```go
type MsgUpdateParams struct {
    // authority is the address that controls the module (defaults to x/gov unless overwritten).
    Authority string `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
    // params defines the x/auth parameters to update.
    //
    // NOTE: All parameters must be supplied.
    Params Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
}
```