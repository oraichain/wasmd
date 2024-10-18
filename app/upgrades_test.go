package app

import (
	"fmt"
	"testing"

	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/stretchr/testify/require"
)

func TestUpgradeIcaController(t *testing.T) {
	app := Setup(t)
	ctx := app.GetBaseApp().NewContext(false)

	testcases := []struct {
		name     string
		malleate func() string
		hasCap   bool
		expPass  bool
	}{
		{
			"success - no chanel",
			func() string {
				return ""
			},
			false,
			true,
		},
		{
			"success - chanels without prefix",
			func() string {
				portIDWithOutPrefix := "port-id"
				chanelId := "chanel-0"
				app.IBCKeeper.ChannelKeeper.SetChannel(ctx, portIDWithOutPrefix, chanelId, channeltypes.Channel{
					ConnectionHops: []string{"connection-0"},
				})

				path := host.ChannelCapabilityPath(portIDWithOutPrefix, chanelId)
				return path
			},
			false,
			true,
		},
		{
			"success - chanels with prefix",
			func() string {
				portIDWithPrefix := fmt.Sprintf("%s%s", icatypes.ControllerPortPrefix, "port-id")
				chanelId := "chanel-0"
				app.IBCKeeper.ChannelKeeper.SetChannel(ctx, portIDWithPrefix, chanelId, channeltypes.Channel{
					ConnectionHops: []string{"connection-0"},
				})

				path := host.ChannelCapabilityPath(portIDWithPrefix, chanelId)
				return path
			},
			true,
			true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(st *testing.T) {
			capPath := tc.malleate()

			err := app.upgradeIcaController(ctx)

			if tc.expPass {
				require.NoError(t, err)

				if tc.hasCap {
					cap, found := app.ScopedICAControllerKeeper.GetCapability(ctx, capPath)
					require.True(t, found)
					require.NotNil(t, cap)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

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
