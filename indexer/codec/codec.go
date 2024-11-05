package codec

import (
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/app/params"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	enccodec "github.com/evmos/ethermint/encoding/codec"
)

func MakeEncodingConfig() params.EncodingConfig {

	cfg := sdk.GetConfig()

	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	cfg.SetAddressVerifier(wasmtypes.VerifyAddressLen())

	encodingConfig := params.MakeEncodingConfig()
	wasmtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	wasmtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	enccodec.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	enccodec.RegisterLegacyAminoCodec(encodingConfig.Amino)

	return encodingConfig
}
