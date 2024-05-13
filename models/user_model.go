package models

type UserToAuth struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type EmailType struct {
	Subject  string `json:"subject"`
	Message  string `json:"message"`
	Receiver string `json:"receiver"`
}
