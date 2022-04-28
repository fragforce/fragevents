package handlers

import "github.com/gin-gonic/gin"

func RegisterHandlers(r *gin.Engine) {
	// Inline handler - just make sure we're alive
	r.GET("/alive", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"alive": true,
			"ok":    true,
			"error": nil,
		})
	})

}
