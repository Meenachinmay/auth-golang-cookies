package handlers

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"strings"
)

func (lac *LocalApiConfig) HandlerPusherAuth(c *gin.Context) {
	// Extract the token from the Authorization header
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization token provided"})
		return
	}

	// Strip 'Bearer ' prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to parse token"})
		return
	}

	// Check if the token was valid
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		_ = claims.UserId // Assuming your Claims struct includes UserID

		// Extract params for Pusher authentication
		params, _ := io.ReadAll(c.Request.Body)
		response, err := lac.PusherClient.AuthorizePrivateChannel(params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication with Pusher failed", "details": err.Error()})
			return
		}

		// Return the auth response to the client
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Write(response)
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
}

func (lac *LocalApiConfig) HandlerCheckWS(c *gin.Context) {
	data := map[string]string{"message": "Hello world"}

	err := lac.PusherClient.Trigger("my-channel", "my-event", data)
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data sent to real time pusher service for the client",
	})
	return

}

func (lac *LocalApiConfig) HandlerSendMessage(c *gin.Context) {
	type NewMessage struct {
		Message  string `json:"message"`
		UserName string `json:"username"`
	}

	message := NewMessage{}
	if err := c.ShouldBindJSON(&message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userIdInterface, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load userId from ginContext.",
		})
		return
	}

	userId, ok := userIdInterface.(uuid.UUID) // Type assertion
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed at type assertion",
		})
		return
	}

	channelName := fmt.Sprintf("private-%s", userId.String())

	err := lac.PusherClient.Trigger(channelName, "new-message", message)
	//err := lac.PusherClient.Trigger("public-chat", "new-message", message)

	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in event triggering from the pusher" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Data sent to real time pusher service for the client",
		"userId":      userId.String(),
		"text":        message,
		"channelName": channelName,
	})
}
