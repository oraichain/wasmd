package utils

import (
	"fmt"

	"github.com/CosmWasm/wasmd/app/params"
	indexerType "github.com/CosmWasm/wasmd/indexer/types"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostx "github.com/cosmos/cosmos-sdk/types/tx"
	signingtx "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/hashicorp/go-hclog"
)

func MarshalSignatureData(signaturesData signingtx.SignatureData, signatures *[][]byte) {
	if sigData, ok := signaturesData.(*signingtx.SingleSignatureData); ok {
		*signatures = append(*signatures, sigData.Signature)
	}
	if multiSigData, ok := signaturesData.(*signingtx.MultiSignatureData); ok {
		for _, sigData := range multiSigData.Signatures {
			MarshalSignatureData(sigData, signatures)
		}
	}
}

func MarshalMsgsAny(config params.EncodingConfig, msgsAny []*types.Any) ([]byte, error) {
	msgsBz := [][]byte{}
	for _, msg := range msgsAny {
		msgMarshal, err := config.Codec.Marshal(msg)
		if err != nil {
			return nil, err
		}
		msgsBz = append(msgsBz, msgMarshal)
	}

	fullMsgsBz, err := config.Amino.Marshal(msgsBz)
	if err != nil {
		return nil, err
	}
	return fullMsgsBz, nil
}

func UnmarshalMsgsBz(config params.EncodingConfig, msgsBz []byte) ([]*types.Any, error) {
	msgsBytes := [][]byte{}
	err := config.Amino.Unmarshal(msgsBz, &msgsBytes)
	if err != nil {
		return nil, err
	}
	msgsAny := []*types.Any{}
	for _, msg := range msgsBytes {
		msgAny := types.Any{}
		err := config.Codec.Unmarshal(msg, &msgAny)
		if err != nil {
			return nil, err
		}
		msgsAny = append(msgsAny, &msgAny)
	}
	return msgsAny, nil
}

func UnmarshalTxBz(indexer indexerType.ModuleEventSinkIndexer, txBz []byte) (*cosmostx.Tx, error) {
	// tx proto
	config := indexer.EncodingConfig()
	tx, err := config.TxConfig.TxDecoder()(txBz)
	if err != nil {
		hclog.Default().Error(fmt.Sprintf("err decoder: %v", err))
		tx, err = config.TxConfig.TxJSONDecoder()(txBz)
		if err != nil {
			panic(err)
		}
	}
	msgs := tx.GetMsgs()
	if err != nil {
		panic(err)
	}

	// try getting memo
	sdkTx := tx.(authsigning.Tx)
	memo := sdkTx.GetMemo()
	timeoutHeight := sdkTx.GetTimeoutHeight()
	granter := sdkTx.FeeGranter()
	payer := sdkTx.FeePayer()
	fees := sdkTx.GetFee()
	gas := sdkTx.GetGas()
	fee := cosmostx.Fee{Amount: fees, GasLimit: gas, Payer: sdk.AccAddress(payer).String(), Granter: sdk.AccAddress(granter).String()}
	msgsAny, err := cosmostx.SetMsgs(msgs)
	if err != nil {
		return nil, err
	}
	body := cosmostx.TxBody{
		Messages:      msgsAny,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}

	pubKeys, err := sdkTx.GetPubKeys()
	if err != nil {
		return nil, err
	}
	signers := []*cosmostx.SignerInfo{}
	for _, pubkey := range pubKeys {
		pubkeyAny, err := types.NewAnyWithValue(pubkey)
		if err != nil {
			return nil, err
		}
		signerInfo := cosmostx.SignerInfo{PublicKey: pubkeyAny}
		signers = append(signers, &signerInfo)
	}
	authInfo := cosmostx.AuthInfo{
		Fee:         &fee,
		SignerInfos: signers,
	}
	signatures := [][]byte{}
	signaturesV2, err := sdkTx.GetSignaturesV2()
	if err != nil {
		return nil, err
	}
	for _, sig := range signaturesV2 {
		MarshalSignatureData(sig.Data, &signatures)
	}
	return &cosmostx.Tx{Body: &body, AuthInfo: &authInfo, Signatures: signatures}, nil
}
