package authz

import (
	"fmt"
	"math/big"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/ethereum/go-ethereum/common"
)

func ParseSetGrantArgs(
	args []interface{},
) (
	grantee common.Address,
	denom string,
	amount *big.Int,
	err error,
) {
	if len(args) != 3 {
		return common.Address{}, "", nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	grantee, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, "", nil, fmt.Errorf("invalid grantee: %v", args[0])
	}

	denom, ok = args[1].(string)
	if !ok {
		return common.Address{}, "", nil, fmt.Errorf("invalid denom: %v", args[1])
	}

	amount, ok = args[2].(*big.Int)
	if !ok {
		return common.Address{}, "", nil, fmt.Errorf("invalid amount: %v", args[2])
	}

	return grantee, denom, amount, nil
}

func ParseExecGrantArgs(
	args []interface{},
) (
	granter common.Address,
	recipient common.Address,
	denom string,
	amount *big.Int,
	err error,
) {
	if len(args) != 4 {
		return common.Address{}, common.Address{}, "", nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	granter, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, common.Address{}, "", nil, fmt.Errorf("invalid granter: %v", args[0])
	}

	recipient, ok = args[1].(common.Address)
	if !ok {
		return common.Address{}, common.Address{}, "", nil, fmt.Errorf("invalid recipient: %v", args[1])
	}

	denom, ok = args[2].(string)
	if !ok {
		return common.Address{}, common.Address{}, "", nil, fmt.Errorf("invalid denom: %v", args[2])
	}

	amount, ok = args[3].(*big.Int)
	if !ok {
		return common.Address{}, common.Address{}, "", nil, fmt.Errorf("invalid amount: %v", args[3])
	}

	return granter, recipient, denom, amount, nil
}

func ParseGetAuthorizationArgs(
	args []interface{},
) (
	granter common.Address,
	grantee common.Address,
	denom string,
	err error,
) {
	if len(args) != 3 {
		return common.Address{}, common.Address{}, "", fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	granter, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, common.Address{}, "", fmt.Errorf("invalid granter: %v", args[0])
	}

	grantee, ok = args[1].(common.Address)
	if !ok {
		return common.Address{}, common.Address{}, "", fmt.Errorf("invalid grantee: %v", args[1])
	}

	denom, ok = args[2].(string)
	if !ok {
		return common.Address{}, common.Address{}, "", fmt.Errorf("invalid denom: %v", args[2])
	}

	return granter, grantee, denom, nil
}
