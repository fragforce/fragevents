package handler_reg

import (
	"github.com/fragforce/fragevents/lib/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterHandlers(r *gin.Engine) {
	// Add more here that should only be used for web hosting

	// Temp stuff
	r.GET("/team/:teamid/", handlers.GetTeam)
	r.POST("/v1/register/:rtype/", handlers.RegisterType)

	// Registration
	r.POST("/v1/:rtype/register", handlers.RegisterType)
	// Cached calls
	r.GET("/v1/team/:teamid/", handlers.GetTeam)

}
