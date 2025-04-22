package tx

import (
	"context"
	"fmt"
	"strconv"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/aminojson"
)

const EIP191MessagePrefix = "\x19Ethereum Signed Message:\n"

// SignModeHandler defines the SIGN_MODE_EIP191 and extends the SIGN_MODE_LEGACY_AMINO_JSON.
type SignModeEIP191Handler struct {
	aminojson.SignModeHandler
}

// NewSignModeHandler returns a new SignModeHandler.
func NewEIP191SignModeHandler(options aminojson.SignModeHandlerOptions) *SignModeEIP191Handler {
	h := &SignModeEIP191Handler{
		SignModeHandler: *aminojson.NewSignModeHandler((options)),
	}

	return h
}

func (h SignModeEIP191Handler) Mode() signingv1beta1.SignMode {
	return signingv1beta1.SignMode_SIGN_MODE_EIP_191
}

func (h SignModeEIP191Handler) GetSignBytes(ctx context.Context, signerData signing.SignerData, txData signing.TxData) ([]byte, error) {

	aminoJSONBz, err := h.SignModeHandler.GetSignBytes(ctx, signerData, txData)

	if err != nil {
		return nil, fmt.Errorf("SignMode_SIGN_MODE_EIP_191 cannot parse into pretty amino json: '%v': '%+v'", string(aminoJSONBz), err)
	}

	bz := append(
		[]byte(EIP191MessagePrefix),
		[]byte(strconv.Itoa(len(aminoJSONBz)))...,
	)

	bz = append(bz, aminoJSONBz...)

	return bz, nil
}
