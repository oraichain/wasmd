package authz

// import (
// 	_ "embed"
// 	"errors"
// 	"fmt"
// 	"math/big"
// 	"strings"

// 	sdkmath "cosmossdk.io/math"
// 	pcommon "github.com/CosmWasm/wasmd/precompile/common"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

// 	"github.com/cosmos/cosmos-sdk/x/authz"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/precompile/contract"
// )

// // Singleton StatefulPrecompiledContract.
// var (
// 	// RawABI contains the raw ABI of wasmd contract.
// 	//go:embed abi.json
// 	RawABI string

// 	ABI = contract.MustParseABI(RawABI)
// )

// const (
// 	SetGrantMethod  = "setGrant"
// 	ExecGrantMethod = "execGrant"
// 	GrantMethod     = "grant"
// )

// const (
// 	NoGrantError = "authorization not found for"
// )

// type PrecompileExecutor struct {
// 	evmKeeper   pcommon.EVMKeeper
// 	authzKeeper pcommon.AuthzKeeper
// }

// // NewContract returns a new authz stateful precompiled contract.
// //
// //	This contract is used for testing purposes only and should not be used on public chains.
// //	The functions of this contract (once implemented), will be used to exercise and test the various aspects of
// //	the EVM such as gas usage, argument parsing, events, etc. The specific operations tested under this contract are
// //	still to be determined.
// func NewContract(evmKeeper pcommon.EVMKeeper, authzKeeper pcommon.AuthzKeeper) contract.StatefulPrecompiledContract {
// 	executor := &PrecompileExecutor{
// 		evmKeeper:   evmKeeper,
// 		authzKeeper: authzKeeper,
// 	}

// 	functions := []*contract.StatefulPrecompileFunction{
// 		contract.NewStatefulPrecompileFunction(
// 			ABI.Methods[SetGrantMethod].ID,
// 			executor.setGrant,
// 		),
// 		contract.NewStatefulPrecompileFunction(
// 			ABI.Methods[ExecGrantMethod].ID,
// 			executor.execGrant,
// 		),
// 		contract.NewStatefulPrecompileFunction(
// 			ABI.Methods[GrantMethod].ID,
// 			executor.grant,
// 		),
// 	}

// 	// Construct the contract with functions.
// 	precompile, err := contract.NewStatefulPrecompileContract(functions)
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to instantiate authz precompile: %s", err.Error()))
// 	}

// 	return precompile
// }

// // Transaction function
// func (p PrecompileExecutor) setGrant(
// 	accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int,
// ) (ret []byte, remainingGas uint64, rerr error) {
// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s", err)
// 			return

// 		}
// 	}()

// 	method := ABI.Methods[SetGrantMethod]
// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if readOnly {
// 		rerr = errors.New("cannot call save grant from staticcall")
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
// 		rerr = err
// 		return
// 	}

// 	granteeAddress := args[0].(common.Address)

// 	denom := args[1].(string)
// 	if denom == "" {
// 		rerr = errors.New("invalid denom")
// 		return
// 	}

// 	amount := args[2].(*big.Int)
// 	if amount.Cmp(big.NewInt(0)) == 0 {
// 		// short circuit
// 		ret, rerr = method.Outputs.Pack(true)
// 		return
// 	}

// 	granterCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)
// 	granteeCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, granteeAddress)
// 	grantCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)))
// 	authorization := banktypes.NewSendAuthorization(grantCoins, []sdk.AccAddress{})

// 	// We consider expire time = nil
// 	setGrantMsg, err := authz.NewMsgGrant(granterCosmosAddr, granteeCosmosAddr, authorization, nil)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	_, err = p.authzKeeper.Grant(ctx, setGrantMsg)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	ret, rerr = method.Outputs.Pack(true)
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)
// 	return
// }

// func (p PrecompileExecutor) execGrant(
// 	accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int,
// ) (ret []byte, remainingGas uint64, rerr error) {
// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s", err)
// 			return

// 		}
// 	}()

// 	method := ABI.Methods[ExecGrantMethod]
// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if readOnly {
// 		rerr = errors.New("cannot call exec grant from staticcall")
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 4); err != nil {
// 		rerr = err
// 		return
// 	}

// 	granterAddress := args[0].(common.Address)
// 	recipientAddress := args[1].(common.Address)

// 	denom := args[2].(string)
// 	if denom == "" {
// 		rerr = errors.New("invalid denom")
// 		return
// 	}

// 	amount := args[3].(*big.Int)
// 	if amount.Cmp(big.NewInt(0)) == 0 {
// 		// short circuit
// 		ret, rerr = method.Outputs.Pack(true)
// 		return
// 	}

// 	granterCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, granterAddress)
// 	granteeCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)
// 	recipientCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, recipientAddress)

// 	bankSendMsg := banktypes.NewMsgSend(
// 		granterCosmosAddr,
// 		recipientCosmosAddr,
// 		sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount))),
// 	)
// 	execGrantMsg := authz.NewMsgExec(granteeCosmosAddr, []sdk.Msg{bankSendMsg})

// 	_, err = p.authzKeeper.Exec(ctx, &execGrantMsg)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	ret, rerr = method.Outputs.Pack(true)
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)
// 	return
// }

// // Query function
// func (p PrecompileExecutor) grant(
// 	accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int,
// ) (ret []byte, remainingGas uint64, rerr error) {
// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s", err)
// 			ctx.Logger().Error("Error querying grant using authz precompile: ", rerr.Error())
// 			return
// 		}
// 	}()

// 	method := ABI.Methods[GrantMethod]

// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
// 		rerr = err
// 		return
// 	}

// 	granterAddress := args[0].(common.Address)
// 	granteeAddress := args[1].(common.Address)

// 	denom := args[2].(string)
// 	if denom == "" {
// 		rerr = errors.New("invalid denom")
// 		return
// 	}

// 	granterCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, granterAddress)
// 	granteeCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, granteeAddress)

// 	// We consider pagination = nil
// 	grantMsg := &authz.QueryGrantsRequest{
// 		Granter:    granterCosmosAddr.String(),
// 		Grantee:    granteeCosmosAddr.String(),
// 		MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
// 		Pagination: nil,
// 	}

// 	res, err := p.authzKeeper.Grants(ctx, grantMsg)
// 	if err != nil {
// 		if strings.Contains(err.Error(), NoGrantError) {
// 			ret, rerr = method.Outputs.Pack(big.NewInt(0))
// 			remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)

// 			return
// 		}

// 		rerr = err
// 		return
// 	}

// 	var sendAuthorization banktypes.SendAuthorization
// 	var grantCoin sdk.Coin

// 	for _, grant := range res.Grants {
// 		sendAuthorization.Unmarshal(grant.Authorization.Value)

// 		for _, coin := range sendAuthorization.SpendLimit {
// 			if coin.Denom == denom {
// 				grantCoin = coin
// 			}
// 		}
// 	}

// 	if grantCoin.Denom == "" {
// 		rerr = errors.New("invalid grant denom")
// 		return
// 	}

// 	ret, rerr = method.Outputs.Pack(grantCoin.Amount.BigInt())
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())

// 	return
// }
