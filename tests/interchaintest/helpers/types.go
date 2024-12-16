package helpers

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ParamChange struct {
	Subspace string `json:"subspace"`
	Key      string `json:"key"`
	Value    any    `json:"value"`
}

// TokenFactory type
type QueryTokenFactoryParamsResponse struct {
	Params TokenFactoryParams `json:"params"`
}

type TokenFactoryParams struct {
	DenomCreationFee        sdk.Coins `json:"denom_creation_fee"`
	DenomCreationGasConsume string    `json:"denom_creation_gas_consume,omitempty"`
}

type QueryDenomsFromCreatorResponse struct {
	Denoms []string `json:"denoms"`
}

type QueryDenomAuthorityMetadataResponse struct {
	AuthorityMetadata DenomAuthorityMetadata `json:"authority_metadata"`
}

type DenomAuthorityMetadata struct {
	Admin string `json:"admin"`
}
