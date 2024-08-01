package v010_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"

	feemarketkeeper "github.com/CosmWasm/wasmd/x/feemarket/keeper"
	v010 "github.com/CosmWasm/wasmd/x/feemarket/migrations/v010"
	v09types "github.com/CosmWasm/wasmd/x/feemarket/migrations/v09/types"

	feemarkettypes "github.com/CosmWasm/wasmd/x/feemarket/types"
)

func TestMigrateStore(t *testing.T) {
	encCfg := wasmkeeper.MakeEncodingConfig(t)
	feemarketKey := storetypes.NewKVStoreKey(feemarkettypes.StoreKey)
	tFeeMarketKey := storetypes.NewTransientStoreKey(fmt.Sprintf("%s_test", feemarkettypes.StoreKey))
	ctx := testutil.DefaultContext(feemarketKey, tFeeMarketKey)
	paramstore := paramtypes.NewSubspace(
		encCfg.Codec, encCfg.Amino, feemarketKey, tFeeMarketKey, "feemarket",
	)
	fmKeeper := feemarketkeeper.NewKeeper(encCfg.Codec, feemarketKey, paramstore)
	fmKeeper.SetParams(ctx, feemarkettypes.DefaultParams())
	require.True(t, paramstore.HasKeyTable())

	// check that the fee market is not nil
	err := v010.MigrateStore(ctx, &paramstore, feemarketKey)
	require.NoError(t, err)
	require.False(t, ctx.KVStore(feemarketKey).Has(v010.KeyPrefixBaseFeeV1))

	params := fmKeeper.GetParams(ctx)
	require.False(t, params.BaseFee.IsNil())

	baseFee := fmKeeper.GetBaseFee(ctx)
	require.NotNil(t, baseFee)

	require.Equal(t, baseFee.Int64(), params.BaseFee.Int64())
}

func TestMigrateJSON(t *testing.T) {
	rawJson := `{
		"base_fee": "669921875",
		"block_gas": "0",
		"params": {
			"base_fee_change_denominator": 8,
			"elasticity_multiplier": 2,
			"enable_height": "0",
			"initial_base_fee": "1000000000",
			"no_base_fee": false
		}
  }`
	encCfg := wasmkeeper.MakeEncodingConfig(t)
	var genState v09types.GenesisState
	err := encCfg.Codec.UnmarshalJSON([]byte(rawJson), &genState)
	require.NoError(t, err)

	migratedGenState := v010.MigrateJSON(genState)

	require.Equal(t, int64(669921875), migratedGenState.Params.BaseFee.Int64())
}
