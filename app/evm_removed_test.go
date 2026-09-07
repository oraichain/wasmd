package app_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	erc20types "github.com/cosmos/evm/x/erc20/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/CosmWasm/wasmd/app"
)

// EVM messages that must stay unreachable for tx execution after soft-removal.
// Type URLs are derived from the real types so assertions cannot pass on a typo.
func removedEVMMsgs() []proto.Message {
	return []proto.Message{
		&evmtypes.MsgEthereumTx{},
		&evmtypes.MsgUpdateParams{},
		&erc20types.MsgConvertERC20{},
		&feemarkettypes.MsgUpdateParams{},
	}
}

// Store keys that MUST stay mounted. The EVM modules are soft-removed: the keepers and
// AppModules are gone but their stores stay in the Multistore so the binary can load
// pre-fork state. Unmounting any of these without a coordinated StoreUpgrades.Deleted
// changes the Multistore layout and breaks the restart.
var mountedEVMStoreKeys = []string{"evm", "feemarket", "erc20", "precisebank"}

// TestEVMMsgsUnreachable is the regression guard for the 0x802 ICS-20 precompile mint
// exploit. Soft-removed EVM msgs must have no msg service route. MsgUpdateParams is
// registered as a decode-only stub (historical gov proposals); other EVM msgs must not
// resolve in the interface registry at all.
func TestEVMMsgsUnreachable(t *testing.T) {
	wasmApp := app.Setup(t)
	router := wasmApp.MsgServiceRouter()
	registry := wasmApp.InterfaceRegistry()

	for _, msg := range removedEVMMsgs() {
		typeURL := "/" + proto.MessageName(msg)

		require.Nil(t, router.HandlerByTypeURL(typeURL),
			"%s must have no msg service route after EVM removal", typeURL)

		_, err := registry.Resolve(typeURL)
		if typeURL == "/cosmos.evm.vm.v1.MsgUpdateParams" {
			// Decode-only stub for historical gov proposals (e.g. proposal 316).
			require.NoError(t, err, "%s must stay registered for gov proposal decode", typeURL)
			continue
		}
		require.Error(t, err,
			"%s must not be registered in the interface registry: a tx carrying it should fail to decode", typeURL)
	}
}

// TestEVMStoreKeysStillMounted fails if someone "cleans up" the soft-removed EVM stores.
func TestEVMStoreKeysStillMounted(t *testing.T) {
	wasmApp := app.Setup(t)

	for _, name := range mountedEVMStoreKeys {
		require.NotNil(t, wasmApp.GetKey(name),
			"store key %q must stay mounted for pre-fork state compatibility", name)
	}
}

// TestEthereumTxHasNoSigners covers the layer below decoding: MsgEthereumTx carries no
// cosmos.msg.v1.signer option and the custom GetSigners resolver was removed with the EVM
// wiring, so even a directly-constructed message cannot resolve a signer.
func TestEthereumTxHasNoSigners(t *testing.T) {
	wasmApp := app.Setup(t)

	_, _, err := wasmApp.AppCodec().GetMsgV1Signers(&evmtypes.MsgEthereumTx{})
	require.Error(t, err, "MsgEthereumTx must not resolve a signer")
}

// TestEthAccountStillDecodable is the counterweight: eth_secp256k1 pubkeys and the legacy
// EthAccount type MUST stay registered, or existing accounts on the chain break.
func TestEthAccountStillDecodable(t *testing.T) {
	wasmApp := app.Setup(t)

	_, err := wasmApp.InterfaceRegistry().Resolve("/cosmos.evm.crypto.v1.ethsecp256k1.PubKey")
	require.NoError(t, err, "eth_secp256k1 pubkeys must stay registered for existing accounts")
}
