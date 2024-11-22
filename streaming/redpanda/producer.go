package redpanda

import (
	"context"
	"encoding/json"
	"time"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/twmb/franz-go/pkg/kgo"
)

type TopicAndKey struct {
	Topic string `json:"topic"`
	Key   string `json:"key"`
}

type Producer struct {
	client *kgo.Client
}

func NewProducer(brokers []string) *Producer {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProduceRequestTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
	}

	return &Producer{client: client}
}

func (p *Producer) SendToRedpanda(topicAndKeys []TopicAndKey, tx ctypes.ResultTx) error {
	ctx := context.Background()
	valueBz, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	for _, topicAndKey := range topicAndKeys {
		topic := topicAndKey.Topic
		key := topicAndKey.Key
		keyBz, err := json.Marshal(key)
		if err != nil {
			return err
		}

		p.client.Produce(ctx, &kgo.Record{Topic: topic, Key: keyBz, Value: valueBz}, func(_ *kgo.Record, e error) {
			err = e
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}
