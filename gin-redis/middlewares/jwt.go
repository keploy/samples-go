package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/keploy/gin-redis/config"
	"github.com/keploy/gin-redis/constants"
	"github.com/keploy/gin-redis/helpers/token"
)

func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("userToken")
		if userToken != "" {
			jwtCheck, err := token.VerifyToken(userToken, config.Get().JWT)
			if err != nil {
				c.AbortWithStatusJSON(401, constants.INVALID_TOKEN_RESPONSE)
				return
			}
			c.Set("email", jwtCheck.Value)
		} else {
			c.AbortWithStatusJSON(401, constants.INVALID_TOKEN_RESPONSE)
		}
	}
}

func SuperJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("userToken")
		if userToken != "" {
			jwtCheck, err := token.VerifyToken(userToken, config.Get().JWT)
			if err != nil {
				c.AbortWithStatusJSON(401, constants.INVALID_TOKEN_RESPONSE)
				return
			}
			if jwtCheck.Value != config.Get().SuperAdmin {
				c.AbortWithStatusJSON(401, constants.INVALID_TOKEN_RESPONSE)
			}
			c.Set("email", jwtCheck.Value)
		} else {
			c.AbortWithStatusJSON(401, constants.INVALID_TOKEN_RESPONSE)
		}
	}
}
