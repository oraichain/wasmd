package sinkreader

import (
	"fmt"
	"os"
)

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
