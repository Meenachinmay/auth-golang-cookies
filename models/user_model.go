package models

type UserToAuth struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
