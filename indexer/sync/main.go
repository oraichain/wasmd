package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/CosmWasm/wasmd/indexer/sink/psql"
	sinkreader "github.com/CosmWasm/wasmd/indexer/sink/reader"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/rpc/client/http"
	cmttypes "github.com/cometbft/cometbft/types"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main", r)
		}
	}()

	// get archive node, start block, end block from cmd
	archiveNode := flag.String("archive-node", "https://rpc.orai.io", "")
	startBlock := flag.Int64("start-block", 0, "")
	endBlock := flag.Int64("end-block", 0, "")
	flag.Parse()

	sinkReader := sinkreader.SinkReaderFromFile{}
	psqlConn, chainId, err := sinkReader.ReadEventSinkInfo()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	eventSink, err := psql.NewEventSink(psqlConn, chainId)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	rpcClient, err := http.New(*archiveNode, "/websocket")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	for curHeight := *startBlock; curHeight <= *endBlock; curHeight++ {
		blockResult, err := rpcClient.BlockResults(context.Background(), &curHeight)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		numTxs := int64(len(blockResult.TxsResults))

		block, err := rpcClient.Block(context.Background(), &curHeight)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		// index block
		eventNewBlockEvents := cmttypes.EventDataNewBlockEvents{
			Height: curHeight,
			NumTxs: numTxs,
			Events: blockResult.FinalizeBlockEvents,
		}
		err = eventSink.IndexBlockEvents(eventNewBlockEvents)
		if err != nil {
			fmt.Printf("failed to index block, height: %d, err: %v\n", curHeight, err)
			panic(err)
		}

		// index txs
		txResults := []*abci.TxResult{}
		for i := int64(0); i < numTxs; i++ {
			txResult := abci.TxResult{
				Height: curHeight,
				Index:  uint32(i),
				Tx:     block.Block.Txs[i],
				Result: *blockResult.TxsResults[i],
				Time:   &block.Block.Time,
			}
			txResults = append(txResults, &txResult)
		}
		err = eventSink.IndexTxEvents(txResults)
		if err != nil {
			fmt.Printf("failed to index block txs, height: %d, err: %v\n", curHeight, err)
			panic(err)
		}
	}
}
