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
	ParamsKey                   = []byte{0x00} // Prefix for params key
	AllowedTokenKeyPrefix       = []byte{0x01} // Key for the fee allowed token lists
	TokenConfigurationKeyPrefix = []byte{0x02} // Key for the token info
	TokenExchangeRateKeyPrefix  = []byte{0x03} // Key for token exchange rate
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
