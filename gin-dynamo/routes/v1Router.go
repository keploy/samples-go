package routes

import (
	PATCH "user-onboarding/controllers/PATCH"
	POST "user-onboarding/controllers/POST"
	"user-onboarding/middlewares"

	"github.com/gin-gonic/gin"
	"go.elastic.co/apm/module/apmgin"
)

func v1Routes(route *gin.RouterGroup) {

	router := gin.New()
	router.Use(apmgin.Middleware(router))

	v1Routes := route.Group("/v1")
	{
		v1Routes.POST("/login", POST.UserLogin)
		v1Routes.POST("/createUser", POST.CreateUser)
		v1Routes.POST("/fetchUserName", POST.FetchUser)
		v1Routes.Use(middlewares.AuthChecker())
		v1Routes.POST("/updateUser", PATCH.UpdateUser)
	}
}
