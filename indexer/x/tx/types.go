package tx

import "time"

type TxEvent struct {
	Height       uint64
	Index        int64
	ChainId      string
	Type         string
	Key          string
	CompositeKey string
	Value        string
	CreatedAt    time.Time
}
