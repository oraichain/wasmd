package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	// ModuleName defines the module name
	ModuleName = "txfees"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// FoundationGroupPolicyAddress is the only account allowed to add/remove
	// txfees blacklist entries via MsgAddBlacklist / MsgRemoveBlacklist /
	// MsgRemoveBlacklistAndRecover.
	//
	// This MUST be the Oraichain Foundation x/group policy account address
	// (not a CW3 contract and not an EOA). Members submit/vote/exec proposals
	// through x/group; the group module authorizes msgs whose signer equals
	// this policy address, then the txfees msg server checks the same value.
	//
	// Replace with the on-chain group policy address after
	// MsgCreateGroupWithPolicy for the foundation group.
	// Local testnet policy (group_id=1). Replace with mainnet foundation policy before release.
	FoundationGroupPolicyAddress = "orai1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsp4a86q"

	// FoundationAddress is an alias for FoundationGroupPolicyAddress.
	FoundationAddress = FoundationGroupPolicyAddress
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
	BlacklistKeyPrefix          = []byte("blacklist")      // KeyPrefixBlacklist defines prefix key for storing blacklist.
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

func GetBlacklistKey(address sdk.AccAddress) []byte {
	return append(BlacklistKeyPrefix, address...)
}
