package redpanda

import (
	"context"
	"encoding/json"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *kgo.Client
}

func NewProducer(brokers []string) *Producer {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
	)
	if err != nil {
		panic(err)
	}

	return &Producer{client: client}
}

func (p *Producer) SendToRedpanda(topics []string, tx ctypes.ResultTx) error {
	// TODO: Need implement this function to
	// send data to redpanda

	ctx := context.Background()
	b, _ := json.Marshal(tx)

	var err error
	for _, topic := range topics {
		p.client.Produce(ctx, &kgo.Record{Topic: topic, Value: b}, func(_ *kgo.Record, e error) {
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
