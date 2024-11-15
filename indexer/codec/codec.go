package codec

import (
	circuittypes "cosmossdk.io/x/circuit/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	feegrant "cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/app/params"
	tokenfactorytypes "github.com/CosmWasm/wasmd/x/tokenfactory/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	clocktypes "github.com/CosmosContracts/juno/v18/x/clock/types"
	globalfeetypes "github.com/CosmosContracts/juno/v18/x/globalfee/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/group"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	proposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
	enccodec "github.com/evmos/ethermint/encoding/codec"
	erc20types "github.com/evmos/ethermint/x/erc20/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
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

	// register interfaces for modules
	authtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	banktypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	stakingtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	minttypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	distrtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	slashingtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	govv1beta1.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	govtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	crisistypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ibcclienttypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ibctransfertypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	icacontrollertypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	icahosttypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	clocktypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	globalfeetypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	packetforwardtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	tokenfactorytypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	evmtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	feemarkettypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	erc20types.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ibctm.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ibcfeetypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	feegrant.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	authz.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ica.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	circuittypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	vestingtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	group.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	upgradetypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	evidencetypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	consensusparamtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	channeltypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ibcconnectiontypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	commitmenttypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	solomachine.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	localhost.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	cosmostx.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	proposaltypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	authtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	banktypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	stakingtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	minttypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	distrtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	slashingtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	govv1beta1.RegisterLegacyAminoCodec(encodingConfig.Amino)
	govtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	crisistypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ibctransfertypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	clocktypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	globalfeetypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	packetforwardtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	evmtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	feemarkettypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	erc20types.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ibcfeetypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	feegrant.RegisterLegacyAminoCodec(encodingConfig.Amino)
	authz.RegisterLegacyAminoCodec(encodingConfig.Amino)
	vestingtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	group.RegisterLegacyAminoCodec(encodingConfig.Amino)
	upgradetypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	evidencetypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	consensusparamtypes.RegisterLegacyAminoCodec(encodingConfig.Amino)
	proposaltypes.RegisterLegacyAminoCodec(encodingConfig.Amino)

	return encodingConfig
}
