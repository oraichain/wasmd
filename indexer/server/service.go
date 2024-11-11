package server

import (
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/indexer/codec"
	"github.com/CosmWasm/wasmd/indexer/server/config"
	"github.com/CosmWasm/wasmd/indexer/sinkreader"
	"github.com/CosmWasm/wasmd/indexer/x/tx"
	"github.com/cometbft/cometbft/rpc/jsonrpc/server"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/rs/cors"

	"cosmossdk.io/log"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
)

// ServerStartTime defines the time duration that the server need to stay running after startup
// for the startup be considered successful
const (
	ServerStartTime = 5 * time.Second
)

var encodingConfig params.EncodingConfig

func init() {
	encodingConfig = codec.MakeEncodingConfig()
}

// StartIndexerService starts the JSON-RPC server
func StartIndexerService(
	clientCtx client.Context,
	logger log.Logger,
) (*http.Server, chan struct{}, error) {

	configPath := filepath.Join(clientCtx.HomeDir, "config")
	indexerConfig, err := indexer.ReadServiceConfig(configPath, config.IndexerFileName, clientCtx.Viper)
	if err != nil {
		return nil, nil, err
	}

	r := http.NewServeMux()
	sinkReader := sinkreader.NewEventSinkReaderFromIndexerService(clientCtx.Viper, clientCtx.ChainID, clientCtx.HomeDir)
	conn, _, err := sinkReader.ReadEventSinkInfo()
	if err != nil {
		return nil, nil, err
	}
	eventSink, err := psql.NewEventSink(conn, clientCtx.ChainID)
	if err != nil {
		return nil, nil, err
	}
	txEventSink := tx.NewTxEventSinkIndexer(eventSink, encodingConfig)
	env := GetRoutes(txEventSink)
	server.RegisterRPCFuncs(r, env, cmtlog.NewNopLogger())

	handlerWithCors := cors.Default()
	if indexerConfig.IService.EnableUnsafeCORS {
		handlerWithCors = cors.AllowAll()
	}

	httpSrv := &http.Server{
		Addr:              indexerConfig.IService.Address,
		Handler:           handlerWithCors.Handler(r),
		ReadHeaderTimeout: indexerConfig.IService.HTTPTimeout,
		ReadTimeout:       indexerConfig.IService.HTTPTimeout,
		WriteTimeout:      indexerConfig.IService.HTTPTimeout,
		IdleTimeout:       indexerConfig.IService.HTTPIdleTimeout,
	}
	httpSrvDone := make(chan struct{}, 1)

	ln, err := Listen(httpSrv.Addr, indexerConfig)
	if err != nil {
		return nil, nil, err
	}

	errCh := make(chan error)
	go func() {
		logger.Info("Starting Indexer RPC server", "address", indexerConfig.IService.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				close(httpSrvDone)
				return
			}

			logger.Error("failed to start Indexer RPC server", "error", err.Error())
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Error("failed to boot Indexer RPC server", "error", err.Error())
		return nil, nil, err
	case <-time.After(ServerStartTime): // assume Indexer RPC server started successfully
	}

	// allocate separate WS connection to Tendermint
	return httpSrv, httpSrvDone, nil
}

// Listen starts a net.Listener on the tcp network on the given address.
// If there is a specified MaxOpenConnections in the config, it will also set the limitListener.
func Listen(addr string, config *config.Config) (net.Listener, error) {
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return ln, err
}
