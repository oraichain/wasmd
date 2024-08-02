package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// Handler defines the core of the state transition function of an application.
type Handler func(ctx cosmostypes.Context, msg cosmostypes.Msg) (*cosmostypes.Result, error)
