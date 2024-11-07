package redpanda

import "github.com/twmb/franz-go/pkg/kgo"

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

func (p *Producer) SendToRedpanda() {
	// TODO: Need implement this function to
	// send data to redpanda
}

func (p *Producer) Close() {
	p.client.Close()
}
