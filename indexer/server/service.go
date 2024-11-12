package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/indexer/codec"
	"github.com/CosmWasm/wasmd/indexer/server/config"
	"github.com/CosmWasm/wasmd/indexer/x/tx"
	"github.com/cometbft/cometbft/rpc/jsonrpc/server"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"

	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cosmos/cosmos-sdk/client"
	sdksvr "github.com/cosmos/cosmos-sdk/server"
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

// StartIndexerService starts the RPC server, this can be called in post setup of start cmd
func StartIndexerService(
	svrCtx *sdksvr.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group, tmNode *node.Node,
) (func(), error) {

	configPath := filepath.Join(clientCtx.HomeDir, "config")
	indexerConfig, err := indexer.ReadServiceConfig(configPath, config.IndexerFileName, clientCtx.Viper)
	if err != nil {
		svrCtx.Logger.Warn(fmt.Sprintf("Couldn't find the %s.toml file with err: %v. The Indexer RPC won't run", err, config.IndexerFileName))
		return func() {}, nil
	}

	r := http.NewServeMux()
	eventSink, err := psql.NewEventSink(svrCtx.Config.TxIndex.PsqlConn, clientCtx.ChainID)
	if err != nil {
		svrCtx.Logger.Warn(fmt.Sprintf("Couldn't create a new connection to the Postgres DB with err: %v. The Indexer RPC won't run", err))
		return func() {}, nil
	}
	txEventSink := tx.NewTxEventSinkIndexer(eventSink, encodingConfig)
	// init node env to setup indexer RPC
	nodeEnv, err := tmNode.ConfigureRPC()
	if err != nil {
		return func() {}, err
	}
	env := GetRoutes(txEventSink, nodeEnv)
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
		return func() {}, err
	}

	errCh := make(chan error)
	go func() {
		svrCtx.Logger.Info("Starting Indexer RPC server", "address", indexerConfig.IService.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				close(httpSrvDone)
				return
			}

			svrCtx.Logger.Error("failed to start Indexer RPC server", "error", err.Error())
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		svrCtx.Logger.Error("failed to boot Indexer RPC server", "error", err.Error())
		return func() {}, err
	case <-time.After(ServerStartTime): // assume Indexer RPC server started successfully
	}

	// defer func at the end
	return func() {
		shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFn()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			svrCtx.Logger.Error("HTTP server shutdown produced a warning", "error", err.Error())
		} else {
			svrCtx.Logger.Info("HTTP server shut down, waiting 5 sec")
			select {
			case <-time.Tick(5 * time.Second):
			case <-httpSrvDone:
			}
		}
	}, nil
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
