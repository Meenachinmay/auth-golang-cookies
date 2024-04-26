package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (lac *LocalApiConfig) HandlerCheckReadiness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Ok",
	})
}
