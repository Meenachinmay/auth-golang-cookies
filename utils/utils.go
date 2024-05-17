package utils

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func CreateKafkaTopic(adminClient *kafka.AdminClient, topicName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results, err := adminClient.CreateTopics(
		ctx,
		[]kafka.TopicSpecification{{
			Topic:             topicName,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}},
	)

	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			return result.Error
		}
	}

	return nil
}
