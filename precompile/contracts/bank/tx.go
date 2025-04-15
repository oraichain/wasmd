package bank

import (
	"errors"
	"math/big"

	sdkmath "cosmossdk.io/math"
	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	tokenfactorytypes "github.com/CosmWasm/wasmd/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func (p Precompile) Send(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
		return nil, err
	}

	receiverEvmAddr := args[0].(common.Address)
	denom := args[1].(string)
	if denom == "" {
		return nil, errors.New("invalid denom")
	}
	amount := args[2].(*big.Int)
	if amount.Cmp(big.NewInt(0)) == 0 {
		// short circuit
		ret, rerr := method.Outputs.Pack(true)
		return ret, rerr
	}

	caller := contract.CallerAddress

	senderCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)
	receiverCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, receiverEvmAddr)

	if err := p.bankKeeper.SendCoins(ctx, senderCosmosAddr, receiverCosmosAddr, sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)))); err != nil {
		return nil, err
	}

	ret, rerr := method.Outputs.Pack(true)
	return ret, rerr
}

func (p Precompile) Burn(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
		return nil, err
	}

	burnFromEvmAddr := args[0].(common.Address)
	denom := args[1].(string)
	if denom == "" {
		return nil, errors.New("invalid denom")
	}
	amount := args[2].(*big.Int)
	if amount.Cmp(big.NewInt(0)) == 0 {
		// short circuit
		ret, rerr := method.Outputs.Pack(true)
		return ret, rerr
	}

	caller := contract.CallerAddress
	coinBurn := sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount))
	burnFromCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, burnFromEvmAddr)
	callerCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)

	if !burnFromCosmosAddr.Equals(callerCosmosAddr) {
		// case caller is not equal burnFrom then check grant of caller and burnFrom, burnFrom = granter and caller = grantee
		// if has grant then burn coin from burnFrom account
		// if not then return error
		// first we get grant of caller and burnFrom account
		// We consider pagination = nil
		grantMsg := &authz.QueryGrantsRequest{
			Granter:    burnFromCosmosAddr.String(),
			Grantee:    callerCosmosAddr.String(),
			MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
			Pagination: nil,
		}

		res, err := p.authzKeeper.Grants(ctx, grantMsg)
		if err != nil {
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
			return nil, errors.New("invalid grant denom")
		}

		// then we exec grant to token-factory module account
		bankSendMsg := banktypes.NewMsgSend(
			burnFromCosmosAddr,
			burnFromCosmosAddr,
			sdk.NewCoins(coinBurn),
		)
		execGrantMsg := authz.NewMsgExec(callerCosmosAddr, []sdk.Msg{bankSendMsg})

		_, err = p.authzKeeper.Exec(ctx, &execGrantMsg)
		if err != nil {
			return nil, err
		}
	}

	// first send coin from account to token-factory module
	if err := p.bankKeeper.SendCoinsFromAccountToModule(ctx, burnFromCosmosAddr, tokenfactorytypes.ModuleName, sdk.NewCoins(coinBurn)); err != nil {
		return nil, err
	}

	// then burn coin from token-factory module
	if err := p.bankKeeper.BurnCoins(ctx, tokenfactorytypes.ModuleName, sdk.NewCoins(coinBurn)); err != nil {
		return nil, err
	}

	ret, rerr := method.Outputs.Pack(true)
	return ret, rerr

}
