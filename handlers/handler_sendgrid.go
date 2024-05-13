package handlers

import (
	"auth-golang-cookies/models"
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

func (lac *LocalApiConfig) HandlerSendEmail(emailType models.EmailType) (Response, error) {
	from := mail.NewEmail("Chinmay anand", "anand.japan896@icloud.com")
	subject := emailType.Subject
	to := mail.NewEmail("Test user", emailType.Receiver)
	plainTextContent := emailType.Message
	htmlContent := `
		<h1>Test user</h1>
		<p>Hello world!</p>
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
