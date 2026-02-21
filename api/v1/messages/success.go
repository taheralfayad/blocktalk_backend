package messages

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func StatusOk(c *gin.Context, message string){
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}
