package types

import "cosmossdk.io/errors"

var (
	ErrInvalidExchangeRate = errors.Register(ModuleName, 1, "invalid exchange rate")
)
