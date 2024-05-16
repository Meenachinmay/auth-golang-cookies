package handlers

import (
	"auth-golang-cookies/internal/config"
	"auth-golang-cookies/internal/database"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type LocalApiConfig struct {
	*config.ApiConfig
	Producer *kafka.Producer
}

func (lac *LocalApiConfig) HandlerGetUser(c *gin.Context) {
	type GetUserParameters struct {
		ID uuid.UUID `json:"id"`
	}
}

func (lac *LocalApiConfig) HandlerCreateUser(c *gin.Context) {
	type CreateUserParameters struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	user := CreateUserParameters{}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	newUser, err := lac.DB.CreateUser(c, database.CreateUserParams{
		ID:        uuid.New(),
		Name:      user.Name,
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Produce a Kafka message
	message, _ := json.Marshal(newUser)
	topic := "user-signups"
	err = lac.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, nil)
	if err != nil {
		fmt.Printf("Produce message failed: %v\n", err)
		return
	}

	c.JSON(http.StatusOK, newUser)
}
