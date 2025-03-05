// Package router have routes defined in it for REST API
package router

// https://stackoverflow.com/a/62608670
// https://stackoverflow.com/questions/62608429/how-to-combine-group-of-routes-in-gin

import (
	"github.com/gin-gonic/gin"
	"github.com/saketV8/book-store-inventory/pkg/handlers"
	"github.com/saketV8/book-store-inventory/pkg/middleware"
	"github.com/saketV8/book-store-inventory/pkg/utils"
)

func SetupPrivateRouter(app *App, superRouterGroup *gin.Engine) {

	// <ROUTER_PREFIX> = /api
	// <ROUTER_PREFIX_VERSION> = /v1

	// fixing linter issue
	// app will be used when we pass data from app to handler :)
	// see example in <public_router.go> SetupPublicRouter()
	_ = app

	routerGroup := superRouterGroup.Group(utils.ROUTER_PREFIX)
	{
		v1 := routerGroup.Group(utils.ROUTER_PREFIX_VERSION)
		v1.Use(middleware.BasicAuthMiddleware())
		{
			v1.GET("/test-private/", handlers.TestAPI)
		}
	}
}
