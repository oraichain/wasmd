package encoding_test

import (
	"math/big"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/tests"
	evmtypes "github.com/CosmWasm/wasmd/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTxEncoding(t *testing.T) {
	addr, key := tests.NewAddrKey()
	signer := tests.NewSigner(key)

	msg := evmtypes.NewTxContract(big.NewInt(1), 1, big.NewInt(10), 100000, nil, big.NewInt(1), big.NewInt(1), []byte{}, nil)
	msg.From = addr.Hex()

	ethSigner := ethtypes.LatestSignerForChainID(big.NewInt(1))
	err := msg.Sign(ethSigner, signer)
	require.NoError(t, err)

	cfg := app.MakeEncodingConfig(t)
	txEncoder := cfg.TxConfig.TxEncoder()
	_, err = txEncoder(msg)
	require.Error(t, err, "encoding failed")

	// FIXME: transaction hashing is hardcoded on Terndermint:
	// See https://github.com/tendermint/tendermint/issues/6539 for reference
	// txHash := msg.AsTransaction().Hash()
	// tmTx := tmtypes.Tx(bz)

	// require.Equal(t, txHash.Bytes(), tmTx.Hash())
}
