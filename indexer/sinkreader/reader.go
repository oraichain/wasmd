package sinkreader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/indexer/server/config"
	"github.com/spf13/viper"
)

type EventSinkReader interface {
	ReadEventSinkInfo() (string, string, error)
	validateSinkInfo(conn, chainId string) bool
}

var _ EventSinkReader = (*SinkReaderFromEnv)(nil)

type SinkReaderFromEnv struct {
}

func (reader SinkReaderFromEnv) ReadEventSinkInfo() (string, string, error) {
	psqlConn := os.Getenv("PSQL_CONN")
	chainId := os.Getenv("CHAIN_ID")
	if !reader.validateSinkInfo(psqlConn, chainId) {
		fmt.Errorf(fmt.Sprintf("Error reading event sink info from env: Invalid %s and %s must not be empty\n", psqlConn, chainId))
	}
	return psqlConn, chainId, nil
}

func (reader SinkReaderFromEnv) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}

type SinkReaderFromFile struct {
}

func (reader SinkReaderFromFile) ReadEventSinkInfo() (string, string, error) {
	return "", "", fmt.Errorf("Not implemented")
}

func (reader SinkReaderFromFile) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}

func NewEventSinkReader() EventSinkReader {
	psqlConn := os.Getenv("PSQL_CONN")
	if psqlConn != "" {
		return SinkReaderFromFile{}
	}
	return SinkReaderFromEnv{}
}

var _ EventSinkReader = (*SinkReaderFromIndexerSerivce)(nil)

type SinkReaderFromIndexerSerivce struct {
	v       *viper.Viper
	chainID string
	homeDir string
}

func NewEventSinkReaderFromIndexerService(v *viper.Viper, chainId, homDir string) EventSinkReader {
	return SinkReaderFromIndexerSerivce{v: v, chainID: chainId, homeDir: homDir}
}

func (reader SinkReaderFromIndexerSerivce) ReadEventSinkInfo() (string, string, error) {
	configPath := filepath.Join(reader.homeDir, "config")
	config, err := indexer.ReadCometBFTConfig(configPath, config.ConfigFileName, reader.v)
	if err != nil {
		return "", "", err
	}
	return config.TxIndex.PsqlConn, reader.chainID, nil
}

func (reader SinkReaderFromIndexerSerivce) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}
