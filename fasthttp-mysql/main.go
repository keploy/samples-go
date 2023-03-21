package main

import (
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
	routes := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/":
			color.Info.Tips("GET /")
			GETindex(ctx)
		case "/movie":
			if ctx.IsPost() {
				color.Info.Tips("POST /movie")
				POSTMovie(ctx)
			} else {
				color.Info.Tips("GET /movie")
				GETmovie(ctx)
			}
		case "/movies":
			color.Info.Tips("GET /movies")
			GETAllMovies(ctx)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}
	color.Info.Tips("Listening on port %s", port)
	log.Fatal(fasthttp.ListenAndServe(":"+port, kMiddleware(routes)))
}
