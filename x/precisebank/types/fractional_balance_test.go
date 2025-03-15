package types_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/precisebank/types"
	"github.com/stretchr/testify/require"
)

func TestConversionFactor_Immutable(t *testing.T) {
	cf1 := types.ConversionFactor()
	origInt64 := cf1.Int64()

	// Get the internal pointer to the big.Int without copying
	internalBigInt := cf1.BigIntMut()

	// Mutate the big.Int -- .Add() mutates in place
	internalBigInt.Add(internalBigInt, big.NewInt(5))
	// Ensure bigInt was actually mutated
	require.Equal(t, origInt64+5, internalBigInt.Int64())

	// Fetch the max amount again
	cf2 := types.ConversionFactor()

	require.Equal(
		t,
		origInt64,
		cf2.Int64(),
		"conversion factor should be immutable",
	)
}

func TestConversionFactor_Copied(t *testing.T) {
	max1 := types.ConversionFactor().BigIntMut()
	max2 := types.ConversionFactor().BigIntMut()

	// Checks that the returned two pointers do not reference the same object
	require.NotSame(t, max1, max2, "max fractional amount should be copied")
}

func TestConversionFactor(t *testing.T) {
	require.Equal(
		t,
		sdkmath.NewInt(1_000_000_000_000),
		types.ConversionFactor(),
		"conversion factor should have 12 decimal points",
	)
}

func TestNewFractionalBalance(t *testing.T) {
	tests := []struct {
		name        string
		giveAddress string
		giveAmount  sdkmath.Int
	}{
		{
			"correctly sets fields",
			"cosmos1qperwt9wrnkg5k9e5gzfgjppzpqur82k6c5a0n",
			sdkmath.NewInt(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := types.NewFractionalBalance(tt.giveAddress, tt.giveAmount)

			require.Equal(t, tt.giveAddress, fb.Address)
			require.Equal(t, tt.giveAmount, fb.Amount)
		})
	}
}

func TestFractionalBalance_Validate(t *testing.T) {
	app.SetSDKConfig()

	tests := []struct {
		name        string
		giveAddress string
		giveAmount  sdkmath.Int
		wantErr     string
	}{
		{
			"valid",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.NewInt(100),
			"",
		},
		{
			"valid - uppercase address",
			"ORAI1NQ6QFKSRV3ASSF9GZ8H8XNH5QGDW9870T22Q53",
			sdkmath.NewInt(100),
			"",
		},
		{
			"valid - min balance",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.NewInt(1),
			"",
		},
		{
			"valid - max balance",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			types.ConversionFactor().SubRaw(1),
			"",
		},
		{
			"invalid - 0 balance",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.NewInt(0),
			"non-positive amount 0",
		},
		{
			"invalid - empty",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.Int{},
			"nil amount",
		},
		{
			"invalid - mixed case address",
			"orai1nq6qfkSrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.NewInt(100),
			"decoding bech32 failed: string not all lowercase or all uppercase",
		},
		{
			"invalid - non-bech32 address",
			"invalid",
			sdkmath.NewInt(100),
			"decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			"invalid - wrong bech32 prefix",
			"cosmos1qperwt9wrnkg5k9e5gzfgjppzpqur82k7gqd8n",
			sdkmath.NewInt(100),
			"invalid Bech32 prefix; expected orai, got cosmos",
		},
		{
			"invalid - negative amount",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.NewInt(-100),
			"non-positive amount -100",
		},
		{
			"invalid - max amount + 1",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			types.ConversionFactor(),
			"amount 1000000000000 exceeds max of 999999999999",
		},
		{
			"invalid - much more than max amount",
			"orai1nq6qfksrv3assf9gz8h8xnh5qgdw9870t22q53",
			sdkmath.NewInt(100000000000_000),
			"amount 100000000000000 exceeds max of 999999999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := types.NewFractionalBalance(tt.giveAddress, tt.giveAmount)
			err := fb.Validate()

			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}
