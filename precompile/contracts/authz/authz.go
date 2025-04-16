package authz

import (
	_ "embed"
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of wasmd contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	AuthzContractAddress = "0x9000000000000000000000000000000000000005"

	// Execute methods
	SetGrantMethod  = "setGrant"
	ExecGrantMethod = "execGrant"

	// Query methods
	GrantMethod = "grant"

	// Define the minumum gas required needed for each method
	SetGrantMethodRequiredGas  = 100_000
	ExecGrantMethodRequiredGas = 100_000
	GrantMethodRequiredGas     = 30_000
)

const (
	NoGrantError = "authorization not found for"
)

type Precompile struct {
	cmn.Precompile
	EVMKeeper   pcommon.EVMKeeper
	AuthzKeeper pcommon.AuthzKeeper
}

func NewPrecompile(
	evmKeeper pcommon.EVMKeeper,
	authzKeeper pcommon.AuthzKeeper,
) (*Precompile, error) {
	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI: ABI,
		},
		EVMKeeper:   evmKeeper,
		AuthzKeeper: authzKeeper,
	}

	p.SetAddress(common.HexToAddress(AuthzContractAddress))

	return p, nil
}

func (p Precompile) Address() common.Address {
	return p.Precompile.Address()
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case SetGrantMethod:
		return SetGrantMethodRequiredGas
	case ExecGrantMethod:
		return ExecGrantMethodRequiredGas
	case GrantMethod:
		return GrantMethodRequiredGas
	}

	return 0
}

// Run executes the precompiled contract addr methods defined in the ABI.
func (p Precompile) Run(
	evm *vm.EVM,
	contract *vm.Contract,
	readOnly bool,
) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case SetGrantMethod:
		bz, err = p.SetGrant(ctx, contract, method, args)
		break
	case ExecGrantMethod:
		bz, err = p.ExecGrant(ctx, contract, method, args)
		break
	case GrantMethod:
		bz, err = p.GetAuthorization(ctx, contract, method, args)
		break
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
		return nil, err
	}

	return bz, nil
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case SetGrantMethod:
	case ExecGrantMethod:
		return true
	default:
		return false
	}

	return false
}

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
