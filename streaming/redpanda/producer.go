package redpanda

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
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
		return fmt.Errorf("[Streaming] failed to marshal tx: %w", err)
	}

	for _, topicAndKey := range topicAndKeys {
		topic := topicAndKey.Topic
		key := topicAndKey.Key
		keyBz, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("[Streaming] failed to marshal key: %w", err)
		}

		p.client.Produce(ctx, &kgo.Record{Topic: topic, Key: keyBz, Value: valueBz}, func(_ *kgo.Record, e error) {
			err = e
		})
		if err != nil {
			return fmt.Errorf("[Streaming] failed to produce tx: %w", err)
		}
	}

	return nil
}

func (p *Producer) SendBlockToRedpanda(topic string, block abci.RequestFinalizeBlock) error {
	ctx := context.Background()
	valueBz, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("[Streaming] failed to marshal block: %w", err)
	}

	p.client.Produce(ctx, &kgo.Record{Topic: topic, Value: valueBz}, func(_ *kgo.Record, e error) {
		err = e
	})
	if err != nil {
		return fmt.Errorf("[Streaming] failed to produce block: %w", err)
	}

	return nil
}

func (p *Producer) SendTxToRedpanda(topic string, tx ctypes.ResultTx) error {
	ctx := context.Background()
	valueBz, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("[Streaming] failed to marshal tx: %w", err)
	}

	p.client.Produce(ctx, &kgo.Record{Topic: topic, Value: valueBz}, func(_ *kgo.Record, e error) {
		err = e
	})
	if err != nil {
		return fmt.Errorf("[Streaming] failed to produce tx: %w", err)
	}

	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}
