package bank

import (
	_ "embed"
	"fmt"
	"math/big"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

var _ vm.PrecompiledContract = &Precompile{}

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of wasmd contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	BankPrecompileAddress = "0x9000000000000000000000000000000000000004"

	GasSend        = 100_000
	GasBalance     = 2_851
	GasAllBalances = 2_851
	GasName        = 3_421
	GasSymbol      = 3_464
	GasDecimals    = 427
	GasSupply      = 2_477
	GasBurn        = 100_000

	SendMethod        = "send"
	BalanceMethod     = "balance"
	AllBalancesMethod = "allBalances"
	NameMethod        = "name"
	SymbolMethod      = "symbol"
	DecimalsMethod    = "decimals"
	SupplyMethod      = "supply"
	BurnMethod        = "burn"
)

type CoinBalance struct {
	Amount *big.Int
	Denom  string
}

type Precompile struct {
	cmn.Precompile
	evmKeeper   pcommon.EVMKeeper
	bankKeeper  pcommon.BankKeeper
	authzKeeper pcommon.AuthzKeeper
}

func NewPrecompile(
	evmKeeper pcommon.EVMKeeper,
	bankKeeper pcommon.BankKeeper,
	authzKeeper pcommon.AuthzKeeper,
) (*Precompile, error) {
	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI: ABI,
		},
		evmKeeper:   evmKeeper,
		bankKeeper:  bankKeeper,
		authzKeeper: authzKeeper,
	}

	// SetAddress defines the address of the bank precompile contract.
	p.SetAddress(common.HexToAddress(BankPrecompileAddress))

	return p, nil
}

func (p Precompile) Address() common.Address {
	return p.Precompile.Address()
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}

	switch method.Name {
	case SendMethod:
		return GasSend
	case BalanceMethod:
		return GasBalance
	case AllBalancesMethod:
		return GasAllBalances
	case NameMethod:
		return GasName
	case SymbolMethod:
		return GasSymbol
	case DecimalsMethod:
		return GasDecimals
	case SupplyMethod:
		return GasSupply
	case BurnMethod:
		return GasBurn
	default:
		return 0
	}
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case SendMethod:
		bz, err = p.Send(ctx, contract, method, args)
	case BalanceMethod:
		bz, err = p.Balance(ctx, contract, method, args)
	case AllBalancesMethod:
		bz, err = p.AllBalances(ctx, contract, method, args)
	case NameMethod:
		bz, err = p.Name(ctx, contract, method, args)
	case SymbolMethod:
		bz, err = p.Symbol(ctx, contract, method, args)
	case DecimalsMethod:
		bz, err = p.Decimals(ctx, contract, method, args)
	case SupplyMethod:
		bz, err = p.Supply(ctx, contract, method, args)
	case BurnMethod:
		bz, err = p.Burn(ctx, contract, method, args)
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

func (p Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case SendMethod:
		return true
	case BurnMethod:
		return true
	default:
		return false
	}
}
