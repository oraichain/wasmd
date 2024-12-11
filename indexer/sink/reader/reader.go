package reader

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
		return "", "", fmt.Errorf("error reading event sink info from env: Invalid %s and %s must not be empty", psqlConn, chainId)
	}
	return psqlConn, chainId, nil
}

func (reader SinkReaderFromEnv) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}
