package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

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

	err := lac.PusherClient.Trigger("my-channel", "my-event", message)

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
}
