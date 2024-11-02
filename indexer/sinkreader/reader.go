package sinkreader

import (
	"fmt"
	"os"
)

type EventSinkReader interface {
	ReadEventSinkInfo() (string, string)
	validateSinkInfo(conn, chainId string) bool
}

var _ EventSinkReader = (*SinkReaderFromEnv)(nil)

type SinkReaderFromEnv struct {
}

func (reader SinkReaderFromEnv) ReadEventSinkInfo() (string, string) {
	psqlConn := os.Getenv("PSQL_CONN")
	chainId := os.Getenv("CHAIN_ID")
	if !reader.validateSinkInfo(psqlConn, chainId) {
		panic(fmt.Sprintf("Error reading event sink info from env: Invalid %s and %s must not be empty\n", psqlConn, chainId))
	}
	return psqlConn, chainId
}

func (reader SinkReaderFromEnv) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}

type SinkReaderFromFile struct {
}

func (reader SinkReaderFromFile) ReadEventSinkInfo() (string, string) {
	panic("Not implemented yet!")
	// return "", ""
}

func (reader SinkReaderFromFile) validateSinkInfo(conn, chainId string) bool {
	return conn != "" && chainId != ""
}

func NewEventSinkReader() EventSinkReader {
	psqlConn := os.Getenv("PSQL_CONN")
	if psqlConn == "" {
		return SinkReaderFromFile{}
	}
	return SinkReaderFromEnv{}
}
