package middlewares

import (
	utils "user-onboarding/utils"

	"github.com/gin-gonic/gin"
)

func AuthChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("token")
		validToken := utils.VerifyToken(token)
		if !validToken {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
