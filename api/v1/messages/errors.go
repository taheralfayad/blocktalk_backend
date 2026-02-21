package messages

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func InternalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

func StatusConflict(c *gin.Context, err error) {
	c.JSON(http.StatusConflict, gin.H{
		"error": err.Error(),
	})
}

func StatusUnauthorized(c *gin.Context, err error) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": err.Error(),
	})
}
