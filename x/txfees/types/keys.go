package types

const (
	// ModuleName defines the module name
	ModuleName = "txfees"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
)

type (
	ByPassMsgKey struct{}
)

var (
	ParamsKey                   = []byte("params_key")     // Prefix for params key
	AllowedTokenKeyPrefix       = []byte("allowed_token")  // Key for the fee allowed token lists
	TokenConfigurationKeyPrefix = []byte("token_config")   // Key for the token info
	TokenExchangeRateKeyPrefix  = []byte("token_exchange") // Key for token exchange rate
	EpochKeyPrefix              = []byte("epoch")          // KeyPrefixEpoch defines prefix key for storing epochs.
	BaseDenomKey                = []byte("base_denom")     // BaseDenomKey
)

func GetAllowedTokenKey(denom string) []byte {
	return append(AllowedTokenKeyPrefix, []byte(denom)...)
}

func GetTokenConfigurationKey(denom string) []byte {
	return append(TokenConfigurationKeyPrefix, []byte(denom)...)
}

func GetTokenExchangeRateKey(denom string) []byte {
	return append(TokenExchangeRateKeyPrefix, []byte(denom)...)
}

func GetEpochKey(identifier string) []byte {
	return append(EpochKeyPrefix, []byte(identifier)...)
}
