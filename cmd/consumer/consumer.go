package main

import (
	"auth-golang-cookies/models"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"net/http"
	"os"
)

type Response struct {
	StatusCode int
	Body       string
	Headers    map[string][]string
}

func main() {
	// Print the SendGrid API Key (ensure it's being set correctly)
	fmt.Println("SENDGRID_API_KEY:", os.Getenv("SENDGRID_API_KEY"))

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "email-consumer",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{"user-signups"}, nil)
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Subscribed to topic: %s\n", err)
	}

	fmt.Printf("Consumer is running...: \n")

	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
			continue
		}

		var newUser models.User
		err = json.Unmarshal(msg.Value, &newUser)
		if err != nil {
			fmt.Printf("Failed to unmarshal message: %v\n", err)
			continue
		}

		response, err := sendWelcomeEmail(newUser.Email)
		if err != nil {
			fmt.Printf("Failed to send welcome email to %s: %v\n", newUser.Email, err)
		} else {
			fmt.Printf("Welcome email sent to %s: %v\n", newUser.Email, response)
		}

	}

	consumer.Close()
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
