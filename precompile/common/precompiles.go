package common

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

/*
address constant WASMD_PRECOMPILE_ADDRESS = 0x9000000000000000000000000000000000000001;
address constant JSON_PRECOMPILE_ADDRESS = 0x9000000000000000000000000000000000000002;
address constant ADDR_PRECOMPILE_ADDRESS = 0x9000000000000000000000000000000000000003;
address constant BANK_PRECOMPILE_ADDRESS = 0x9000000000000000000000000000000000000004;
address constant AUTHZ_PRECOMPILE_ADDRESS = 0x9000000000000000000000000000000000000005;
*/

func ValidateArgsLength(args []interface{}, length int) error {
	if len(args) != length {
		return fmt.Errorf("expected %d arguments but got %d", length, len(args))
	}

	return nil
}

func ValidateNonPayable(value *big.Int) error {
	if value != nil && value.Sign() != 0 {
		return errors.New("sending funds to a non-payable function")
	}

	return nil
}

func GetPrecompileCtx(accessibleState contract.AccessibleState) (sdk.Context, uint64, error) {
	stateDB, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		return sdk.UnwrapSDKContext(context.Background()), 0, errors.New("cannot get context from EVM")
	}

	ctx := stateDB.Ctx()
	initialGas := ctx.GasMeter().GasConsumed()
	return ctx, initialGas, nil
}
