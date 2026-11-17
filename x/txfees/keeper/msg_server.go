package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) AddFeeToken(goCtx context.Context, msg *types.MsgAddFeeToken) (*types.MsgAddFeeTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	isAllowed, _ := k.IsTokenAllowed(ctx, msg.Denom)
	if isAllowed {
		return nil, errors.Wrapf(types.ErrTokenAllowed, "fee token allowed %s", msg.Denom)
	}

	k.AddAllowedToken(ctx, msg.Denom)

	return &types.MsgAddFeeTokenResponse{}, nil
}

func (k msgServer) RemoveFeeToken(goCtx context.Context, msg *types.MsgRemoveFeeToken) (*types.MsgRemoveFeeTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	isAllowed, _ := k.IsTokenAllowed(ctx, msg.Denom)
	if !isAllowed {
		return nil, errors.Wrapf(types.ErrTokenAllowed, "fee token not allowed %s", msg.Denom)
	}

	k.RemoveAllowedToken(ctx, msg.Denom)
	return &types.MsgRemoveFeeTokenResponse{}, nil
}

func (k msgServer) AddBlacklist(goCtx context.Context, msg *types.MsgAddBlacklist) (*types.MsgAddBlacklistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := assertBlacklistAuthority(msg.Authority); err != nil {
		return nil, err
	}

	addr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid blacklist address %s", msg.Address)
	}

	if err := k.assertBlacklistable(ctx, addr); err != nil {
		return nil, err
	}

	already, err := k.IsBlacklisted(ctx, addr)
	if err != nil {
		return nil, err
	}
	if already {
		return nil, errors.Wrapf(types.ErrAddressAlreadyBlacklisted, "%s", msg.Address)
	}

	// Intended execution path: x/group proposal whose policy account is
	// FoundationGroupPolicyAddress, with this msg as proposal content. The group
	// module ensures msg signers == policy account before routing here.
	if err := k.Keeper.AddBlacklist(ctx, addr); err != nil {
		return nil, errors.Wrapf(err, "add blacklist %s", msg.Address)
	}
	return &types.MsgAddBlacklistResponse{}, nil
}

func (k msgServer) RemoveBlacklist(goCtx context.Context, msg *types.MsgRemoveBlacklist) (*types.MsgRemoveBlacklistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := assertBlacklistAuthority(msg.Authority); err != nil {
		return nil, err
	}

	addr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid blacklist address %s", msg.Address)
	}

	blacklisted, err := k.IsBlacklisted(ctx, addr)
	if err != nil {
		return nil, err
	}
	if !blacklisted {
		return nil, errors.Wrapf(types.ErrAddressNotBlacklisted, "%s", msg.Address)
	}

	if err := k.Keeper.RemoveBlacklist(ctx, addr); err != nil {
		return nil, errors.Wrapf(err, "remove blacklist %s", msg.Address)
	}
	return &types.MsgRemoveBlacklistResponse{}, nil
}

func (k msgServer) RemoveBlacklistAndRecover(goCtx context.Context, msg *types.MsgRemoveBlacklistAndRecover) (*types.MsgRemoveBlacklistAndRecoverResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := assertBlacklistAuthority(msg.Authority); err != nil {
		return nil, err
	}

	fromAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid blacklist address %s", msg.Address)
	}
	toAddr, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid recover to address %s", msg.ToAddress)
	}

	blacklisted, err := k.IsBlacklisted(ctx, fromAddr)
	if err != nil {
		return nil, err
	}
	if !blacklisted {
		return nil, errors.Wrapf(types.ErrAddressNotBlacklisted, "%s", msg.Address)
	}

	// Recovering into a blacklisted destination would re-freeze the funds the
	// moment they land, with no in-module way back out.
	toBlacklisted, err := k.IsBlacklisted(ctx, toAddr)
	if err != nil {
		return nil, err
	}
	if toBlacklisted {
		return nil, errors.Wrapf(types.ErrAddressAlreadyBlacklisted, "recover destination %s", msg.ToAddress)
	}

	// Remove first so bank send-restriction no longer blocks outbound from fromAddr.
	if err := k.Keeper.RemoveBlacklist(ctx, fromAddr); err != nil {
		return nil, errors.Wrapf(err, "remove blacklist %s", msg.Address)
	}

	amounts := k.bankKeeper.SpendableCoins(ctx, fromAddr)
	if !amounts.IsZero() {
		if err := k.bankKeeper.SendCoins(ctx, fromAddr, toAddr, amounts); err != nil {
			return nil, errors.Wrapf(types.ErrRecoverFailed, "send from %s to %s: %v", msg.Address, msg.ToAddress, err)
		}
	}

	return &types.MsgRemoveBlacklistAndRecoverResponse{Amounts: amounts}, nil
}

// assertBlacklistable rejects addresses that must never be blacklisted.
//
//   - Module accounts: the bank send restriction in app.RegisterBankSendRestrictions
//     applies to SendCoinsFromModuleTo* as well, so blacklisting e.g. fee_collector
//     or bonded_tokens_pool makes the distribution/staking BeginBlocker error and
//     panic every block — an unrecoverable halt, since no tx (not even
//     MsgRemoveBlacklist) can be processed any more.
//   - The foundation policy account itself: blacklisting it would freeze the only
//     authority able to undo the entry.
func (k msgServer) assertBlacklistable(ctx sdk.Context, addr sdk.AccAddress) error {
	if types.IsBlacklistAuthority(addr.String()) {
		return errors.Wrapf(
			types.ErrUnblacklistableAddress,
			"%s is the foundation x/group policy account", addr.String(),
		)
	}

	if k.accKeeper == nil {
		return nil
	}

	if acc := k.accKeeper.GetAccount(ctx, addr); acc != nil {
		if _, isModule := acc.(sdk.ModuleAccountI); isModule {
			return errors.Wrapf(
				types.ErrUnblacklistableAddress,
				"%s is a module account", addr.String(),
			)
		}
	}

	// A module account that has never been instantiated has no entry in the auth
	// store, so GetAccount above returns nil for it. Match on the registered
	// module permissions too.
	for name, perm := range k.accKeeper.GetModulePermissions() {
		if perm.GetAddress().Equals(addr) {
			return errors.Wrapf(
				types.ErrUnblacklistableAddress,
				"%s is the %s module account", addr.String(), name,
			)
		}
	}

	return nil
}

func assertBlacklistAuthority(authority string) error {
	if !types.IsBlacklistAuthority(authority) {
		return errors.Wrapf(
			govtypes.ErrInvalidSigner,
			"invalid authority; expected foundation x/group policy %s, got %s",
			types.FoundationGroupPolicyAddress,
			authority,
		)
	}
	return nil
}
