package handler_global

import (
	"github.com/gin-gonic/gin"
)

func RegisterGlobalHandlers(r *gin.Engine) {
	// FIXME: Make not inline - Inline handler - just make sure we're alive
	r.GET("/alive", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"alive": true,
			"ok":    true,
			"error": nil,
		})
	})

	// Add more here that should be used for groupcache, web, etc
}
