package handlers

import "github.com/gin-gonic/gin"

func RegisterHandlers(r *gin.Engine) {
	RegisterGlobalHandlers(r)
	// Add more here that should only be used for web hosting

}

func RegisterGlobalHandlers(r *gin.Engine) {
	// Inline handler - just make sure we're alive
	r.GET("/alive", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"alive": true,
			"ok":    true,
			"error": nil,
		})
	})

	// Add more here that should be used for groupcache, web, etc
}
