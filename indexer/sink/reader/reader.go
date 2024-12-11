package reader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CosmWasm/wasmd/app"
	idxServer "github.com/CosmWasm/wasmd/server"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config"
	ClientFileName = "client"
)

type EventSinkReader interface {
	ReadEventSinkInfo() (string, string, error)
	validateSinkInfo(string, string) bool
}

var _ EventSinkReader = (*SinkReaderFromEnv)(nil)

type SinkReaderFromEnv struct {
}

func (reader SinkReaderFromEnv) ReadEventSinkInfo() (string, string, error) {
	psqlConn := os.Getenv("PSQL_CONN")
	chainId := os.Getenv("CHAIN_ID")
	if !reader.validateSinkInfo(psqlConn, chainId) {
		return "", "", fmt.Errorf("error reading event sink info from env: Invalid %s and %s must not be empty", psqlConn, chainId)
	}

	return psqlConn, chainId, nil
}

func (reader SinkReaderFromEnv) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}

var _ EventSinkReader = (*SinkReaderFromFile)(nil)

type SinkReaderFromFile struct {
}

func (reader SinkReaderFromFile) ReadEventSinkInfo() (string, string, error) {
	v := viper.New()
	homePath := os.Getenv("HOME_PATH")
	if homePath == "" {
		homePath = app.DefaultNodeHome
	}

	configPath := filepath.Join(homePath, "config")
	config, err := idxServer.ReadCometBFTConfig(configPath, ConfigFileName, v)
	if err != nil {
		return "", "", fmt.Errorf("can not read cometbft config file with error: %v", err)
	}
	psqlConn := config.TxIndex.PsqlConn

	clientCfg, err := idxServer.ReadClientConfig(configPath, ClientFileName, v)
	if err != nil {
		return "", "", fmt.Errorf("can not read client config file with error: %v", err)
	}
	chainId := clientCfg.ChainID

	if !reader.validateSinkInfo(psqlConn, chainId) {
		return "", "", fmt.Errorf("error reading event sink info from env: Invalid %s and %s must not be empty", psqlConn, chainId)
	}

	return psqlConn, chainId, nil
}

func (reader SinkReaderFromFile) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}
