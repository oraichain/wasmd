package eip712

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func getTypedDataDomain(chainID uint64) apitypes.TypedDataDomain {
	return apitypes.TypedDataDomain{
		Name:              "Kava Cosmos",
		Version:           "1.0.0",
		ChainId:           math.NewHexOrDecimal256(int64(chainID)),
		VerifyingContract: "kavaCosmos",
		Salt:              "0",
	}
}

func getRootTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": {
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "version",
				Type: "string",
			},
			{
				Name: "chainId",
				Type: "uint256",
			},
			{
				Name: "verifyingContract",
				Type: "string",
			},
			{
				Name: "salt",
				Type: "string",
			},
		},
		"Tx": {
			{Name: "account_number", Type: "string"},
			{Name: "chain_id", Type: "string"},
			{Name: "fee", Type: "Fee"},
			{Name: "memo", Type: "string"},
			{Name: "sequence", Type: "string"},
			// Note timeout_height was removed because it was not getting filled with the legacyTx
			// {Name: "timeout_height", Type: "string"},
		},
		"Fee": {
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		},
		"Coin": {
			{Name: "denom", Type: "string"},
			{Name: "amount", Type: "string"},
		},
	}
}
