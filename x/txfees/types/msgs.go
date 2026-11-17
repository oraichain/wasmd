package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgAddBlacklist{}
	_ sdk.Msg = &MsgRemoveBlacklist{}
	_ sdk.Msg = &MsgRemoveBlacklistAndRecover{}
)

func (m *MsgAddBlacklist) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid blacklist address (%s)", err)
	}
	return nil
}

func (m *MsgRemoveBlacklist) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid blacklist address (%s)", err)
	}
	return nil
}

func (m *MsgRemoveBlacklistAndRecover) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}
	fromAddr, err := sdk.AccAddressFromBech32(m.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid blacklist address (%s)", err)
	}
	toAddr, err := sdk.AccAddressFromBech32(m.ToAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid recover to address (%s)", err)
	}
	if fromAddr.Equals(toAddr) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "blacklist address and recover to address must differ")
	}
	return nil
}

// IsBlacklistAuthority reports whether authority is the foundation x/group policy.
// Compares AccAddress bytes so Bech32 string form must decode to the same account.
func IsBlacklistAuthority(authority string) bool {
	authAddr, err := sdk.AccAddressFromBech32(authority)
	if err != nil {
		return false
	}
	expected, err := sdk.AccAddressFromBech32(FoundationGroupPolicyAddress)
	if err != nil {
		return false
	}
	return authAddr.Equals(expected)
}
