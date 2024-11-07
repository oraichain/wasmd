package redpanda

import (
	"os"
	"strings"

	indexerConfig "github.com/CosmWasm/wasmd/indexer/config"
)

type RedpandaInfo struct {
	brokers  []string
	topic    string
	admin    *Admin
	producer *Producer
}

func (ri *RedpandaInfo) SetBrokers() {
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

func (ri *RedpandaInfo) GetBrockers() []string {
	return ri.brokers
}

func (ri *RedpandaInfo) SetTopic(module indexerConfig.IndexerModule) {
	topic := os.Getenv("REDPANDA_TOPIC_" + string(module))
	if topic == "" {
		panic("Topic must not be empty")
	}

	ri.topic = topic
}

func (ri *RedpandaInfo) GetTopic() string {
	return ri.topic
}

func (ri *RedpandaInfo) SetAdmin() {
	admin := NewAdmin(ri.brokers)
	ri.admin = admin
}

func (ri *RedpandaInfo) GetAdmin() *Admin {
	return ri.admin
}

func (ri *RedpandaInfo) SetProducer() {
	producer := NewProducer(ri.brokers, ri.topic)
	ri.producer = producer
}

func (ri *RedpandaInfo) GetProducer() *Producer {
	return ri.producer
}
