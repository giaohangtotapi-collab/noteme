package utils

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, data gin.H) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    data,
	})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{
		"success": false,
		"error":   msg,
	})
}
