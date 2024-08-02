package evm

import (
	errorsmod "cosmossdk.io/errors"
	apptypes "github.com/CosmWasm/wasmd/types"
	"github.com/CosmWasm/wasmd/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewHandler returns a handler for Ethermint type messages.
func NewHandler(server types.MsgServer) apptypes.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (result *sdk.Result, err error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *types.MsgEthereumTx:
			// execute state transition
			res, err := server.EthereumTx(ctx, msg)
			return sdk.WrapServiceResult(ctx, res, err)

		default:
			err := errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, err
		}
	}
}
