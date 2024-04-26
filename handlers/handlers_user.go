package handlers

import (
	"auth-golang-cookies/internal/config"
	"auth-golang-cookies/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type LocalApiConfig struct {
	*config.ApiConfig
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

	c.JSON(http.StatusOK, newUser)
}
