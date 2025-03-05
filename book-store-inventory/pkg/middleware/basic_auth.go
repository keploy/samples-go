// Package middleware contains basic user and pass middleware for testing private routes
package middleware

import "github.com/gin-gonic/gin"

func BasicAuthMiddleware() gin.HandlerFunc {

	return gin.BasicAuth(gin.Accounts{
		"saket":  "1234",
		"maurya": "1234",
	})
}
