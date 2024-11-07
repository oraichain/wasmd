package redpanda

import (
	"context"
	"fmt"

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
	)
	if err != nil {
		panic(err)
	}

	admin := kadm.NewClient(client)

	return &Admin{client: admin}
}

func (a *Admin) IsTopicExist(ctx context.Context, topic string) bool {
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

func (a *Admin) CreateTopic(ctx context.Context, topic string) error {
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
