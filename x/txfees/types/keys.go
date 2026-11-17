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
	// DO NOT CHANGE. Verified against Oraichain mainnet live state on
	// 2026-11-17 (chain-id Oraichain, queried at height 130470210):
	//
	//	group_id             1          (the only group on chain)
	//	metadata             "Foundation Operator"
	//	decision_policy      ThresholdDecisionPolicy, threshold 4
	//	members              6, weight 1 each (total_weight 6) -> 4-of-6
	//	voting_period        24h
	//	min_execution_period 0s
	//	admin                this policy address itself (self-governed)
	//	created_at           2026-11-17T04:34:24Z
	//
	// The address is derived from the x/group policy account sequence
	// (keeper.groupPolicySeq -> authtypes.NewModuleCredential), which is
	// monotonic and never reset, so sequence 1 is permanently consumed by this
	// policy and the address cannot be claimed by anyone else.
	//
	// Because admin is the policy itself, the decision policy can only be
	// changed by a 4-of-6 proposal, and x/gov cannot override this authority.
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
