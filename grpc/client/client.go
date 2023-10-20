package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

type application struct {
	g pb.GreeterClient
}

func (app *application) grpcHandler(c echo.Context) error {
	// The grpc connection is in the handler.
	fmt.Println("grpc handler called")
	//	Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Hour)
	defer cancel()
	var wg sync.WaitGroup
	var goR1, goR2 string
	wg.Add(1)
	go func() {
		defer wg.Done()
		goR1 = "req: go-Routine1"
		r, err := app.g.SayHello(ctx, &pb.HelloRequest{Name: "go-Routine1"})
		if err != nil {
			log.Printf("could not greet: %v", err)
			//return c.String(http.StatusInternalServerError, err.Error())
		} else {
			goR1 = goR1 + "; Resp : " + r.GetMessage()
			log.Printf("Greeting: %s", r.GetMessage())
		}
	}()
	wg.Add(1)
	go func() {
		goR2 = "req: go-Routine2"
		defer wg.Done()
		r, err := app.g.SayHello(ctx, &pb.HelloRequest{Name: "go-Routine2"})
		if err != nil {
			log.Printf("could not greet: %v", err)
			//return c.String(http.StatusInternalServerError, err.Error())
		} else {
			goR2 = goR2 + "; Resp : " + r.GetMessage()
			log.Printf("Greeting: %s", r.GetMessage())
		}
	}()
	wg.Wait()
	fmt.Println(fmt.Sprintln(goR1+"\n"+goR2))
	return c.String(http.StatusOK, fmt.Sprintln(goR1+"\n"+goR2))

}

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	g := pb.NewGreeterClient(conn)
	app := &application{g: g}

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())

	// Routes
	e.GET("/pingGRPC", app.grpcHandler)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
