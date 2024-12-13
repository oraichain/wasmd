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

func NewRedPandaInfo(brokers []string, topics []string) *RedpandaInfo {
	info := &RedpandaInfo{}
	info.SetBrokers(brokers)
	info.SetTopics(topics...)
	info.SetAdmin()
	info.SetProducer()
	return info
}

func DefaultTopics() []string {
	wasmTopic := "REDPANDA_TOPIC_" + strings.ToUpper(string(wasmtypes.ModuleName))
	bankTopic := "REDPANDA_TOPIC_" + strings.ToUpper(string(banktypes.ModuleName))
	return []string{wasmTopic, bankTopic}
}

func (ri *RedpandaInfo) SetBrokers(initialBrokers []string) {
	if len(initialBrokers) != 0 {
		ri.brokers = initialBrokers
		return
	}
	var brokers []string
	brokersEnv := os.Getenv("REDPANDA_BROKERS")

	for _, broker := range strings.Split(brokersEnv, ",") {
		brokers = append(brokers, strings.TrimSpace(broker))
	}
	if len(brokers) == 0 {
		panic("Length of brokers must greater than 0")
	}

	ri.brokers = brokers
}

func (ri *RedpandaInfo) GetBrokers() []string {
	return ri.brokers
}

func (ri *RedpandaInfo) SetTopics(topics ...string) {
	if len(topics) == 0 {
		defaultTopics := DefaultTopics()
		ri.topics = append(ri.topics, defaultTopics...)

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
