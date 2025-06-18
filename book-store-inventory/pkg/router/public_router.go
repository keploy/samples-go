// https://stackoverflow.com/a/62608670
// https://stackoverflow.com/questions/62608429/how-to-combine-group-of-routes-in-gin

package router

import (
	"github.com/gin-gonic/gin"
	"github.com/saketV8/book-store-inventory/pkg/utils"
)

func SetupPublicRouter(app *App, superRouterGroup *gin.Engine) {

	// <ROUTER_PREFIX> = /api
	// <ROUTER_PREFIX_VERSION> = /v1
	routerGroup := superRouterGroup.Group(utils.ROUTER_PREFIX)
	{
		v1 := routerGroup.Group(utils.ROUTER_PREFIX_VERSION)
		{
			// GET
			v1.GET("/books/", app.Books.GetBooksHandler)
			// v1.GET("/book/id/:book_id", app.Books.GetBookByIDHandler)
			v1.GET("/book/id/*book_id", app.Books.GetBookByIDHandler)

			// POST
			v1.POST("/book/add/", app.Books.AddBookHandler)

			// DELETE
			v1.DELETE("/book/delete/", app.Books.DeleteBookHandler)

			// PATCH
			v1.PATCH("/book/update/", app.Books.UpdateBookHandler)
		}
	}
}
