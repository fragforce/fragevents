package handler_reg

import (
	"github.com/fragforce/fragevents/lib/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterHandlers(r *gin.Engine) {
	// Add more here that should only be used for web hosting

	// Quick GetTeam f
	r.GET("/team/:teamid/", handlers.GetTeam)

}
