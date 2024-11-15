package app

import (
	"testing"

	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"
)

func TestUpgradeMintParams(t *testing.T) {
	app := Setup(t)
	ctx := app.GetBaseApp().NewContext(false)
	mintSpace, _ := app.ParamsKeeper.GetSubspace(minttypes.ModuleName)

	testcases := []struct {
		name     string
		melleate func()
		minGtMax bool
		expPass  bool
	}{
		{
			"success - inflation min less than inflation max",
			func() {
				mintParams := minttypes.DefaultParams()
				mintSpace.SetParamSet(ctx, &mintParams)
			},
			false,
			true,
		},
		{
			"success - inflation min greater than inflation max",
			func() {
				mintParams := minttypes.DefaultParams()
				mintParams.InflationMin = mintParams.InflationMax.Add(mintParams.InflationMin)
				mintSpace.SetParamSet(ctx, &mintParams)
			},
			true,
			true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(st *testing.T) {
			tc.melleate()

			err := app.upgradeMintParams(ctx)

			if tc.expPass {
				require.NoError(t, err)

				if tc.minGtMax {
					var mintParams minttypes.Params
					mintSpace.GetParamSet(ctx, &mintParams)

					require.True(t, mintParams.InflationMin.Equal(mintParams.InflationMax))
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}
