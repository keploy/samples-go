package main

import (
	"fasthttp-sql/models"
	"log"

	"github.com/gookit/color"
	"github.com/keploy/go-sdk/integrations/kfasthttp"
	"github.com/keploy/go-sdk/keploy"
	"github.com/valyala/fasthttp"
)

var port = "8080"

func main() {
	kConfig := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "fasthttp-mysql",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})

	kMiddleware := kfasthttp.FastHttpMiddleware(kConfig)
	db := models.Connect()
	defer db.Close()
	db.SetConnMaxIdleTime(0)

	routes := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/":
			color.Info.Tips("GET /")
			GETindex(ctx, db)
		case "/movie":
			if ctx.IsPost() {
				color.Info.Tips("POST /movie")
				POSTMovie(ctx, db)
			} else {
				color.Info.Tips("GET /movie")
				GETmovie(ctx, db)
			}
		case "/movies":
			color.Info.Tips("GET /movies")
			GETAllMovies(ctx, db)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}

	color.Info.Tips("Listening on port %s", port)
	log.Fatal(fasthttp.ListenAndServe(":"+port, kMiddleware(routes)))
}
