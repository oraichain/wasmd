package wasmd

import (
	_ "embed"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func (p Precompile) instantiateCosmWasm(
	ctx sdk.Context,
	method *abi.Method,
	caller common.Address,
	codeID uint64,
	adminAddr sdk.AccAddress,
	msg []byte,
	label string,
	deposit sdk.Coins,
) ([]byte, error) {
	creator := p.EVMKeeper.GetCosmosAddressMapping(ctx, caller)

	addr, data, err := p.WasmKeeper.Instantiate(ctx, codeID, creator, adminAddr, msg, label, deposit)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(addr.String(), data)
}

func (p Precompile) executeCosmWasm(
	ctx sdk.Context,
	method *abi.Method,
	caller common.Address,
	contractAddr sdk.AccAddress,
	msg []byte,
	deposit sdk.Coins,
) ([]byte, error) {
	senderAddr := p.EVMKeeper.GetCosmosAddressMapping(ctx, caller)

	exeRes, err := p.WasmKeeper.Execute(ctx, contractAddr, senderAddr, msg, deposit)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(exeRes)
}

func (p Precompile) queryCosmWasm(
	ctx sdk.Context,
	method *abi.Method,
	contractAddr sdk.AccAddress,
	req []byte,
) ([]byte, error) {
	queryRes, err := p.WasmKeeper.QuerySmart(ctx, contractAddr, req)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(queryRes)
}
