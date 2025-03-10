package types

import "cosmossdk.io/errors"

var (
	ErrInvalidExchangeRate = errors.Register(ModuleName, 1, "invalid exchange rate")
	ErrTokenAllowed        = errors.Register(ModuleName, 2, "fee token allowed")
)
