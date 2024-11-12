package redpanda

import (
	"os"
	"strings"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type RedpandaInfo struct {
	brokers  []string
	topics   []string
	admin    *Admin
	producer *Producer
}

func (ri *RedpandaInfo) SetBrokers() {
	var brokers []string
	brokersEnv := os.Getenv("REDPANDA_BROKERS")
	if brokersEnv == "" {
		panic("Empty REDPANDA_BROKERS env!")
	}

	for _, broker := range strings.Split(brokersEnv, ",") {
		brokers = append(brokers, strings.TrimSpace(broker))
	}
	if len(brokers) == 0 {
		panic("Length of brokers must greater than 0")
	}

	ri.brokers = brokers
}

func (ri *RedpandaInfo) GetBrockers() []string {
	return ri.brokers
}

func (ri *RedpandaInfo) SetTopics(topics ...string) {
	if len(topics) == 0 {
		wasmTopic := "REDPANDA_TOPIC_" + strings.ToUpper(string(wasmtypes.ModuleName))
		bankTopic := "REDPANDA_TOPIC_" + strings.ToUpper(string(banktypes.ModuleName))
		topics = []string{wasmTopic, bankTopic}
		ri.topics = append(ri.topics, topics...)

		return
	}

	for _, topic := range topics {
		if topic == "" {
			panic("Topic must not empty")
		}

		ri.topics = append(ri.topics, "REDPANDA_TOPIC_"+strings.ToUpper(topic))
	}
}

func (ri *RedpandaInfo) GetTopic() []string {
	return ri.topics
}

func (ri *RedpandaInfo) SetAdmin() {
	admin := NewAdmin(ri.brokers)
	ri.admin = admin
}

func (ri *RedpandaInfo) GetAdmin() *Admin {
	return ri.admin
}

func (ri *RedpandaInfo) SetProducer() {
	producer := NewProducer(ri.brokers)
	ri.producer = producer
}

func (ri *RedpandaInfo) GetProducer() *Producer {
	return ri.producer
}
