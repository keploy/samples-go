// Package routes manage the user-related routes in the application
package routes

import (
	"users-profile/controllers"

	"github.com/gin-gonic/gin"
)

func UserRoute(router *gin.Engine) {
	router.POST("/user", controllers.CreateAUser())
	router.GET("/user/:userId", controllers.GetAUser())
	router.PUT("/user/:userId", controllers.EditAUser())
	router.DELETE("/user/:userId", controllers.DeleteAUser())
	router.GET("/users", controllers.GetAllUsers())
}
