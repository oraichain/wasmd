package v050_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app"
	v0501 "github.com/CosmWasm/wasmd/app/upgrades/v0501"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"
)

func TestUpgradeMintParams(t *testing.T) {
	app := app.Setup(t)
	ctx := app.GetBaseApp().NewContext(false)
	mintSpace, _ := app.ParamsKeeper.GetSubspace(minttypes.ModuleName)

	testcases := []struct {
		name     string
		melleate func()
	}{
		{
			"success - inflation min less than inflation max",
			func() {
				mintParams := minttypes.DefaultParams()
				mintSpace.SetParamSet(ctx, &mintParams)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(st *testing.T) {
			tc.melleate()

			err := v0501.UpgradeMintParams(ctx, &app.ParamsKeeper, &app.MintKeeper)
			require.NoError(t, err)

			var mintParams minttypes.Params
			mintSpace.GetParamSet(ctx, &mintParams)

			require.True(t, mintParams.InflationMin.Equal(mintParams.InflationMax))
			require.True(t, mintParams.InflationMin.Equal(math.LegacyMustNewDecFromStr("0.085")))

			params, err := app.MintKeeper.Params.Get(ctx)
			require.NoError(t, err)
			require.True(t, params.InflationMin.Equal(mintParams.InflationMax))
			require.True(t, params.InflationMin.Equal(math.LegacyMustNewDecFromStr("0.085")))
		})
	}
}
