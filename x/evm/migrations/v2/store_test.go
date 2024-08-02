package v2_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/app"
	v2 "github.com/CosmWasm/wasmd/x/evm/migrations/v2"
	v2types "github.com/CosmWasm/wasmd/x/evm/migrations/v2/types"
	"github.com/CosmWasm/wasmd/x/evm/types"
)

func TestMigrateStore(t *testing.T) {

	encCfg := app.MakeEncodingConfig(t)
	kvStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	tStoreKey := storetypes.NewTransientStoreKey(fmt.Sprintf("%s_test", types.StoreKey))
	ctx := testutil.DefaultContext(kvStoreKey, tStoreKey)
	paramstore := paramtypes.NewSubspace(
		encCfg.Codec, encCfg.Amino, kvStoreKey, tStoreKey, "evm",
	).WithKeyTable(v2types.ParamKeyTable())
	params := v2types.DefaultParams()
	paramstore.SetParamSet(ctx, &params)

	require.Panics(t, func() {
		var result []types.EIP712AllowedMsg
		paramstore.Get(ctx, types.ParamStoreKeyEIP712AllowedMsgs, &result)
	})

	paramstore = paramtypes.NewSubspace(
		encCfg.Codec, encCfg.Amino, kvStoreKey, tStoreKey, "evm",
	).WithKeyTable(types.ParamKeyTable())
	err := v2.MigrateStore(ctx, &paramstore)
	require.NoError(t, err)

	var result []types.EIP712AllowedMsg
	paramstore.Get(ctx, types.ParamStoreKeyEIP712AllowedMsgs, &result)
	require.Equal(t, v2.NewAllowedMsgs, result)
}
