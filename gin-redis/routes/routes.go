package routes

import (
	"github.com/gin-gonic/gin"
	get "github.com/keploy/gin-redis/controllers/GET"
	post "github.com/keploy/gin-redis/controllers/POST"
	"github.com/keploy/gin-redis/helpers/redis"
)

func NewRouter() *gin.Engine {
	redis.RedisInit()
	router := gin.New()
	v1 := router.Group("/api")

	{
		v1.GET("/getVerificationCode", get.VerificationCode)
		v1.POST("/verifyCode", post.VerifyCode)
	}
	return router
}
