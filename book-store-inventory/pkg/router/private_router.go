// https://stackoverflow.com/a/62608670
// https://stackoverflow.com/questions/62608429/how-to-combine-group-of-routes-in-gin

package router

import (
	"github.com/gin-gonic/gin"
	"github.com/saketV8/book-store-inventory/pkg/middleware"
	"github.com/saketV8/book-store-inventory/pkg/utils"
)

func SetupPrivateRouter(app *App, superRouterGroup *gin.Engine) {

	// <ROUTER_PREFIX> = /api
	// <ROUTER_PREFIX_VERSION> = /v1

	routerGroup := superRouterGroup.Group(utils.ROUTER_PREFIX)
	{
		v1 := routerGroup.Group(utils.ROUTER_PREFIX_VERSION)
		v1.Use(middleware.BasicAuthMiddleware())
		{
			// v1.GET("/test-private-api/", handlers.TestApi)
		}
	}
}
