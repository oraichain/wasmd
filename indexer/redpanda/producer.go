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

func (p *Producer) SendToRedpanda(height int64) error {
	// TODO: Need implement this function to
	// send data to redpanda

	ctx := context.Background()
	b, _ := json.Marshal(Message{Height: height})

	var err error
	p.client.Produce(ctx, &kgo.Record{Topic: p.topic, Value: b}, func(_ *kgo.Record, e error) {
		err = e
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}
