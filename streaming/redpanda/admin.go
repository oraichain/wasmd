package redpanda

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Admin struct {
	client *kadm.Client
}

func NewAdmin(brokers []string) *Admin {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProduceRequestTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
	}

	admin := kadm.NewClient(client)

	return &Admin{client: admin}
}

func (a *Admin) IsTopicExist(topic string) bool {
	ctx := context.Background()
	topicMetadatas, err := a.client.ListTopics(ctx)
	if err != nil {
		panic(err)
	}

	for _, metadata := range topicMetadatas {
		if metadata.Topic == topic {
			return true
		}
	}

	return false
}

func (a *Admin) CreateTopic(topic string) error {
	ctx := context.Background()
	res, err := a.client.CreateTopics(ctx, 1, 1, nil, topic)
	if err != nil {
		return err
	}

	for _, ctr := range res {
		if ctr.Err != nil {
			return errors.Errorf(fmt.Sprintf("unable to create topic %s: %s", ctr.Topic, ctr.Err))
		} else {
			hclog.Default().Info("created topic %s", ctr.Topic)
		}
	}

	return nil
}

func (a *Admin) Close() {
	a.client.Close()
}
