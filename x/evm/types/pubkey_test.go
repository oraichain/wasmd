package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPubkeyToEVMAddress(t *testing.T) {
	pubkey := "Ah4NweWyFaVG5xcOwY5I7Tm4mmfPgLtS+Qn3jvXLX0VP"
	actualEvmAddress, err := PubkeyToEVMAddress(pubkey)
	require.NoError(t, err)
	require.Equal(t, "0x39D8810d16Bc6E8888F78E7F01D8B9999CE03499", actualEvmAddress.Hex())
}

func TestPubkeyToCosmosAddress(t *testing.T) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("orai", "oraipub")
	pubkey := "ApmMmUYx5+SmJ5I+VYufU2qDeaEleQClfRm6sAcXNXvQ"
	actualCosmosAddress, err := PubkeyToCosmosAddress(pubkey)
	require.NoError(t, err)
	require.Equal(t, "orai1sm9hjeczmfxfvgzsdtqt7a7zrtqvkdqarwehg4", actualCosmosAddress.String())
}
