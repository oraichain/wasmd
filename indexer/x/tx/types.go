package tx

import (
	"time"
)

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

type EventAttribute struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Index bool   `protobuf:"varint,3,opt,name=index,proto3" json:"index,omitempty"`
}

type Event struct {
	Type       string           `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Attributes []EventAttribute `protobuf:"bytes,2,rep,name=attributes,proto3" json:"attributes,omitempty"`
}