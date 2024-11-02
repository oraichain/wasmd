package emitter

import abci "github.com/cometbft/cometbft/abci/types"

type StreamingEventEmitter interface {
	EmitModuleEvents(req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error
}

var _ StreamingEventEmitter = (*KafkaEventEmitter)(nil)

type KafkaEventEmitter struct {
}

func NewEventEmitter() StreamingEventEmitter {
	return KafkaEventEmitter{}
}

func (kee KafkaEventEmitter) EmitModuleEvents(req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	panic("Not implemented!")
}
