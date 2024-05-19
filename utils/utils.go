package utils

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"log"
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
		if result.Error.Code() == kafka.ErrTopicAlreadyExists {
			log.Printf("Topic '%s' already exists", topicName)
			return nil
		} else if result.Error.Code() != kafka.ErrNoError {
			log.Printf("Failed to create topic '%s': %v", topicName, result.Error)
			return result.Error
		}
	}

	return nil
}

// CreateKafkaTopics creates multiple Kafka topics if they don't exist
func CreateKafkaTopics(adminClient *kafka.AdminClient, topics []string) error {
	var topicSpecifications []kafka.TopicSpecification
	for _, topic := range topics {
		topicSpecifications = append(topicSpecifications, kafka.TopicSpecification{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	// Create topics
	results, err := adminClient.CreateTopics(nil, topicSpecifications)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			log.Printf("Failed to create topic %s: %v", result.Topic, result.Error)
		} else {
			log.Printf("Created topic %s", result.Topic)
		}
	}
	return nil
}
