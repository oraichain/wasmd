package server

import (
	"github.com/CosmWasm/wasmd/indexer/x/tx"
	rpc "github.com/cometbft/cometbft/rpc/jsonrpc/server"
)

// TODO: better system than "unsafe" prefix

type RoutesMap map[string]*rpc.RPCFunc

// Routes is a map of available routes.
func GetRoutes(cs *tx.TxEventSink) RoutesMap {
	return RoutesMap{
		// info AP
		// "tx":        rpc.NewRPCFunc(env.Tx, "hash,prove", rpc.Cacheable()),
		"tx_search": rpc.NewRPCFunc(cs.TxSearch, "query,limit"),
	}
}
