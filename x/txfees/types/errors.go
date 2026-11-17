package types

import "cosmossdk.io/errors"

var (
	ErrInvalidExchangeRate        = errors.Register(ModuleName, 1, "invalid exchange rate")
	ErrTokenAllowed               = errors.Register(ModuleName, 2, "fee token allowed")
	ErrTooManyFeeCoins            = errors.Register(ModuleName, 3, "too many fee coins. only accepts fees in one denom")
	ErrFeeTokenUnAvailable        = errors.Register(ModuleName, 4, "fee token unavailable")
	ErrTokenConfigurationNotFound = errors.Register(ModuleName, 5, "fee token configuration not found")
	ErrAddressNotBlacklisted      = errors.Register(ModuleName, 6, "address is not blacklisted")
	ErrAddressAlreadyBlacklisted  = errors.Register(ModuleName, 7, "address is already blacklisted")
	ErrRecoverFailed              = errors.Register(ModuleName, 8, "failed to recover assets from blacklisted address")
	ErrUnblacklistableAddress     = errors.Register(ModuleName, 9, "address cannot be blacklisted")
)
