// https://stackoverflow.com/a/62608670
// https://stackoverflow.com/questions/62608429/how-to-combine-group-of-routes-in-gin

package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/saketV8/book-store-inventory/pkg/handlers"
	"github.com/saketV8/book-store-inventory/pkg/utils"

	log "github.com/sirupsen/logrus"
)

type App struct {
	// Add more handlers as needed
	Books *handlers.BookHandler
}

// func SetupRouter(DbModel *querydb.DbModel) {
func SetupRouter(app *App) {
	// Initializing the GIN Router
	gin.SetMode(gin.ReleaseMode)
	rtr := gin.Default()

	SetupPublicRouter(app, rtr)
	SetupPrivateRouter(app, rtr)

	fmt.Println("Setting up gin router 😉")
	fmt.Println()
	fmt.Println("Application is ready 🚀")
	fmt.Println()
	fmt.Println("🔥 Try at http://localhost:9090/api/v1/")
	fmt.Println()
	// print list of all available routes
	utils.ListAllAvailableRoutes(rtr)

	// starting the server
	// <PORT> = :9090
	err := rtr.Run(utils.PORT)
	if err != nil {
		fmt.Println("============================================")
		log.Fatal("ERROR: while Initializing the server")
		fmt.Println("============================================")
	}
}
