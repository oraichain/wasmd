package bank

import (
	"errors"
	"math/big"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	"github.com/ethereum/go-ethereum/common"
)

func parseTxArgs(args []interface{}) (common.Address, string, *big.Int, error) {
	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
		return common.Address{}, "", nil, err
	}

	receiverEvmAddr := args[0].(common.Address)
	denom := args[1].(string)
	if denom == "" {
		return common.Address{}, "", nil, errors.New("invalid denom")
	}
	amount := args[2].(*big.Int)
	if amount.Cmp(big.NewInt(0)) == 0 {
		return common.Address{}, "", nil, errors.New("invalid amount")
	}

	return receiverEvmAddr, denom, amount, nil
}

func parseBalanceArgs(args []interface{}) (common.Address, string, error) {
	if err := pcommon.ValidateArgsLength(args, 2); err != nil {
		return common.Address{}, "", err
	}
	evmAddr := args[0].(common.Address)
	denom := args[1].(string)
	if denom == "" {
		return common.Address{}, "", errors.New("invalid denom")
	}
	return evmAddr, denom, nil
}

func parseAllBalancesArgs(args []interface{}) (common.Address, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return common.Address{}, err
	}
	evmAddr := args[0].(common.Address)
	if evmAddr == (common.Address{}) {
		return common.Address{}, errors.New("invalid evm address")
	}
	return evmAddr, nil
}

func parseErc20Args(args []interface{}) (string, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return "", err
	}
	denom := args[0].(string)
	if denom == "" {
		return "", errors.New("invalid denom")
	}
	return denom, nil
}
