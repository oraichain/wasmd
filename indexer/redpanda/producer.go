package redpanda

import (
	"context"
	"encoding/json"

	"github.com/twmb/franz-go/pkg/kgo"
)

// FIXME: sample message to send to redpanda
type Message struct {
	Height int64 `json:"height"`
}

type Producer struct {
	client *kgo.Client
	topic  string
}

func NewProducer(brokers []string, topic string) *Producer {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
	)
	if err != nil {
		panic(err)
	}

	return &Producer{client: client, topic: topic}
}

func (p *Producer) SendToRedpanda(ctx context.Context, height int64) {
	// TODO: Need implement this function to
	// send data to redpanda

	b, _ := json.Marshal(Message{Height: height})

	p.client.Produce(ctx, &kgo.Record{Topic: p.topic, Value: b}, nil)
}

func (p *Producer) Close() {
	p.client.Close()
}
