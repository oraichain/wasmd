package authz

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p Precompile) GetAuthorization(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	granter, grantee, denom, err := ParseGetAuthorizationArgs(args)
	if err != nil {
		return nil, err
	}

	if denom == "" {
		return nil, fmt.Errorf("invalid denom: %v", denom)
	}

	granterCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, granter)
	granteeCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, grantee)

	// We consider pagination = nil
	grantMsg := &authz.QueryGrantsRequest{
		Granter:    granterCosmosAddr.String(),
		Grantee:    granteeCosmosAddr.String(),
		MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
		Pagination: nil,
	}

	res, err := p.AuthzKeeper.Grants(ctx, grantMsg)
	if err != nil {
		if strings.Contains(err.Error(), NoGrantError) {
			return method.Outputs.Pack(big.NewInt(0))
		}

		return nil, err
	}

	var sendAuthorization banktypes.SendAuthorization
	var grantCoin sdk.Coin

	for _, grant := range res.Grants {
		sendAuthorization.Unmarshal(grant.Authorization.Value)

		for _, coin := range sendAuthorization.SpendLimit {
			if coin.Denom == denom {
				grantCoin = coin
			}
		}
	}

	if grantCoin.Denom == "" {
		return nil, fmt.Errorf("invalid grant denom: %v", denom)
	}

	return method.Outputs.Pack(grantCoin.Amount.BigInt())
}
