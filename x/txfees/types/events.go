package types

const (
	EventTypeEpochEnd       = "epoch_end" // TODO: need to clean up (not use)
	EventTypeEpochStart     = "epoch_start"
	AttributeEpochNumber    = "epoch_number"
	AttributeEpochStartTime = "start_time"
)

// Blacklist events. Every blacklist state transition emits one, so the freeze
// history of an account is reconstructable from block results alone without
// replaying state. MsgRemoveBlacklistAndRecover emits both
// EventTypeRemoveBlacklist and EventTypeRecoverBlacklist: it really does
// unfreeze the account, so a monitor watching only "unfrozen" must see it too.
const (
	EventTypeAddBlacklist     = "add_blacklist"
	EventTypeRemoveBlacklist  = "remove_blacklist"
	EventTypeRecoverBlacklist = "recover_blacklist"

	AttributeKeyAddress   = "address"
	AttributeKeyToAddress = "to_address"
	AttributeKeyAmounts   = "amounts"
	AttributeKeyAuthority = "authority"
)
