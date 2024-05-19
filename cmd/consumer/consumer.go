package main

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Response struct {
	StatusCode int
	Body       string
	Headers    map[string][]string
}

const maxRetries = 5
const retryDelay = 10 * time.Second

var producer *kafka.Producer

func main() {
	// Print the SendGrid API Key (ensure it's being set correctly)
	fmt.Println("SENDGRID_API_KEY:", os.Getenv("SENDGRID_API_KEY"))

	//// initialize Kafka producer
	//var err error
	//producer, err = kafka.NewProducer(&kafka.ConfigMap{
	//	"bootstrap.servers": "localhost:9092",
	//})
	//if err != nil {
	//	log.Fatalf("failed creating producer: %s", err)
	//}
	//defer producer.Close()
	//
	//// initialize the kafka admin client for producer
	//adminClientProducer, err := kafka.NewAdminClientFromProducer(producer)
	//if err != nil {
	//	log.Fatalf("failed to create admin client: %s", err)
	//}
	//defer adminClientProducer.Close()
	//
	//// initialize the kafka admin client
	//adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
	//	"bootstrap.servers": "localhost:9092",
	//})
	//if err != nil {
	//	log.Fatalf("failed to create admin client: %s", err)
	//}
	//defer adminClient.Close()
	//
	//// Create topics if they don't exist
	//topics := []string{"user-signups", "auth-requests", "auth-responses"}
	//err = CreateKafkaTopics(adminClient, topics)
	//if err != nil {
	//	log.Fatalf("Failed to create topics: %s", err)
	//}

	////create the topic if it doesn't exist
	//topic := "auth-requests"
	//err = utils.CreateKafkaTopic(adminClient, topic)
	//if err != nil {
	//	log.Fatalf("failed to create topic: %s", err)
	//}
	//topic2 := "auth-responses"
	//err = utils.CreateKafkaTopic(adminClient, topic2)
	//if err != nil {
	//	log.Fatalf("failed to create topic: %s", err)
	//}

	// Initialize Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "auth-service-consumer",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// initializing kafka consumer admin client
	adminClientConsumer, err := kafka.NewAdminClientFromConsumer(consumer)
	if err != nil {
		fmt.Printf("failed to create admin client: %s\n", err)
		os.Exit(1)
	}
	defer adminClientConsumer.Close()

	// Subscribe to multiple topics
	err = consumer.SubscribeTopics([]string{"new-user-signup"}, nil)
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Consumer is running...: \n")
		fmt.Printf("Subscribed to topic: %s\n", err)
	}

	// get the list of all topics
	topicMetadata, err := adminClientConsumer.GetMetadata(nil, true, 10000)
	if err != nil {
		fmt.Printf("failed to get metadata: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("all topics in the cluster: ")
	for _, topic := range topicMetadata.Topics {
		fmt.Println(topic.Topic)
	}

	// Run the consumer in a separate go routine
	go consumeMessages(consumer)

	//	success := false
	//	for retries := 0; retries < maxRetries; retries++ {
	//		response, err := sendWelcomeEmail(newUser.Email)
	//
	//		if err != nil {
	//			fmt.Printf("Failed to send welcome email to %s: %v\n", newUser.Email, err)
	//			time.Sleep(retryDelay)
	//		} else {
	//			fmt.Printf("Welcome email sent to %s: %v\n", newUser.Email, response)
	//			success = true
	//			break
	//		}
	//	}
	//	if !success {
	//		fmt.Printf("Failed to send welcome email to %s after %d attempts\n: %v\n", newUser.Email, maxRetries, err)
	//	}
	// }

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan

	fmt.Println("shutting down gracefully")
}

func consumeMessages(consumer *kafka.Consumer) {
	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			fmt.Printf("consumer error %v (%v)\n", err, msg)
			continue
		}

		// handle messages based on topic
		switch *msg.TopicPartition.Topic {
		case "new-user-signup":
			email := string(msg.Value)
			success := false
			for retries := 0; retries < maxRetries; retries++ {
				_, err := sendWelcomeEmail(email)

				if err != nil {
					fmt.Printf("Failed to send welcome email to %s: %v: retrying...\n", email, err)
					time.Sleep(retryDelay)
				} else {
					success = true
					break
				}
			}
			if !success {
				fmt.Printf("Failed to send welcome email to %s after %d attempts\n: %v\n", email, maxRetries, err)
			} else {
				fmt.Printf("Sent welcome email to %s\n", email)
			}
		}
		//case "auth-requests":
		//	handleAuthRequests(msg)
	}
}

func sendWelcomeEmail(email string) (Response, error) {
	from := mail.NewEmail("Chinmay anand", "anand.japan896@icloud.com")
	subject := "Welcome"
	to := mail.NewEmail("Test user", email)
	plainTextContent := "Welcome New User"
	htmlContent := `
		<h1>Welcome User Email</h1>
		<p>Hello User</p>
	`
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Fatalln(err)
		return Response{}, err
	}

	sendResponse := Response{
		StatusCode: response.StatusCode,
		Body:       response.Body,
		Headers:    convertHeaders(response.Headers),
	}

	return sendResponse, nil
}

func convertHeaders(headers http.Header) map[string][]string {
	result := map[string][]string{}

	for key, values := range headers {
		result[key] = values
	}
	return result
}
