package legacy

import (
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// IsSubset will return true if the caller is the same as the superset,
// or if the caller is more restrictive than the superset.
func (a AccessTypeV1Beta1) IsSubset(superSet AccessTypeV1Beta1) bool {
	switch superSet {
	case AccessTypeEverybodyV1Beta1:
		// Everything is a subset of this
		return a != AccessTypeUnspecifiedV1Beta1
	case AccessTypeNobodyV1Beta1:
		// Only an exact match is a subset of this
		return a == AccessTypeNobodyV1Beta1
	case AccessTypeAnyOfAddressesV1Beta1:
		// Nobody or address(es)
		return a == AccessTypeNobodyV1Beta1 || a == AccessTypeAnyOfAddressesV1Beta1
	default:
		return false
	}
}

// IsSubset will return true if the caller is the same as the superset,
// or if the caller is more restrictive than the superset.
func (a AccessConfigV1Beta1) IsSubset(superSet AccessConfigV1Beta1) bool {
	return false
}

// AllAuthorizedAddresses returns the list of authorized addresses. Can be empty.
func (a AccessConfigV1Beta1) AllAuthorizedAddresses() []string {
	if a.Permission == AccessTypeAnyOfAddressesV1Beta1 {
		return a.Addresses
	}
	return []string{}
}

// ValidateBasic performs basic validation
func (a AccessConfigV1Beta1) ValidateBasic() error {
	return types.ErrEmpty
}
