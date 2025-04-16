package authz

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p Precompile) SetGrant(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, denom, amount, err := ParseSetGrantArgs(args)
	if err != nil {
		return nil, err
	}

	if denom == "" {
		return nil, fmt.Errorf("invalid denom: %v", denom)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		// short circuit
		return method.Outputs.Pack(true)
	}

	caller := contract.CallerAddress

	granterCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, caller)
	granteeCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, grantee)
	grantCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)))
	authorization := banktypes.NewSendAuthorization(grantCoins, []sdk.AccAddress{})

	// We consider expire time = nil
	setGrantMsg, err := authz.NewMsgGrant(granterCosmosAddr, granteeCosmosAddr, authorization, nil)
	if err != nil {
		return nil, err
	}

	_, err = p.AuthzKeeper.Grant(ctx, setGrantMsg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) ExecGrant(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	granter, recipient, denom, amount, err := ParseExecGrantArgs(args)
	if err != nil {
		return nil, err
	}

	if denom == "" {
		return nil, fmt.Errorf("invalid denom: %v", denom)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		// short circuit
		return method.Outputs.Pack(true)
	}

	caller := contract.CallerAddress

	granterCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, granter)
	granteeCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, caller)
	recipientCosmosAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, recipient)

	bankSendMsg := banktypes.NewMsgSend(
		granterCosmosAddr,
		recipientCosmosAddr,
		sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount))),
	)
	execGrantMsg := authz.NewMsgExec(granteeCosmosAddr, []sdk.Msg{bankSendMsg})

	_, err = p.AuthzKeeper.Exec(ctx, &execGrantMsg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
