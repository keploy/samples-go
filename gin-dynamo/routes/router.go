package routes

import (
	"user-onboarding/middlewares"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {

	router := gin.New()
	router.Use(middlewares.CORSMiddleware()) //removing cors error for interacting with frontend

	v1 := router.Group("/api")
	v1Routes(v1)

	return router
}
