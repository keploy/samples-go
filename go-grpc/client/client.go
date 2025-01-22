// NO NEED
package main

import (
	"context"
	"io"
	"log"
	"net/http"

	pb "github.com/keploy/samples-go/go-grpc/user"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcClient pb.UserServiceClient
)

func init() {

	// Set up the gRPC connection
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}
	grpcClient = pb.NewUserServiceClient(conn)
}

func createUser(c *gin.Context) {
	var userRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int32  `json:"age"`
	}

	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := &pb.UserRequest{
		Name:  userRequest.Name,
		Email: userRequest.Email,
		Age:   userRequest.Age,
	}

	// Call the CreateUser gRPC method
	res, err := grpcClient.CreateUser(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func getUsers(c *gin.Context) {
	// Call the GetUsers gRPC method
	res, err := grpcClient.GetUsers(context.Background(), &pb.Empty{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func updateUser(c *gin.Context) {
	var userRequest struct {
		ID    int32  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int32  `json:"age"`
	}

	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := &pb.UserRequest{
		Id:    userRequest.ID,
		Name:  userRequest.Name,
		Email: userRequest.Email,
		Age:   userRequest.Age,
	}

	// Call the UpdateUser gRPC method
	res, err := grpcClient.UpdateUser(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func deleteUser(c *gin.Context) {
	var userID struct {
		ID int32 `json:"id"`
	}

	// Parse the JSON payload from the HTTP request
	if err := c.ShouldBindJSON(&userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req := &pb.UserID{
		Id: userID.ID,
	}

	// Call the DeleteUser gRPC method
	_, err := grpcClient.DeleteUser(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func createUsersStream(c *gin.Context) {
	// Client streaming: Accept a series of user data and send them to the gRPC server
	stream, err := grpcClient.CreateUsersStream(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var users []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int32  `json:"age"`
	}

	if err := c.ShouldBindJSON(&users); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, user := range users {
		req := &pb.UserRequest{
			Name:  user.Name,
			Email: user.Email,
			Age:   user.Age,
		}

		if err := stream.Send(req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Receive the response from the server
	res, err := stream.CloseAndRecv()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func deleteUsersStream(c *gin.Context) {
	// Client streaming: Accept a series of user data and send them to the gRPC server
	stream, err := grpcClient.DeleteUsersStream(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var userIDs []struct {
		ID int32 `json:"id"`
	}

	// Parse the JSON payload from the HTTP request
	if err := c.ShouldBindJSON(&userIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	for _, user := range userIDs {
		req := &pb.UserID{
			Id: user.ID,
		}

		if err := stream.Send(req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Receive the response from the server
	res, err := stream.CloseAndRecv()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func getUsersStream(c *gin.Context) {
	// Server streaming: Receive users from gRPC server stream
	stream, err := grpcClient.GetUsersStream(context.Background(), &pb.Empty{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var users []pb.User
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		users = append(users, pb.User{Id: user.User.GetId(),
			Name:  user.User.GetName(),
			Email: user.User.GetEmail(),
			Age:   user.User.GetAge(),
		})
	}

	c.JSON(http.StatusOK, users)
}

func UpdateUsersStream(c *gin.Context) {
	// Duplex streaming: Send and receive user updates concurrently
	stream, err := grpcClient.UpdateUsersStream(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var userRequests []struct {
		ID    int32  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int32  `json:"age"`
	}

	// Parse incoming user data
	if err := c.ShouldBindJSON(&userRequests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updUsers []pb.User
	done := make(chan bool)

	// Goroutine for sending data
	go func() {
		for _, userRequest := range userRequests {
			req := &pb.UserRequest{
				Id:    userRequest.ID,
				Name:  userRequest.Name,
				Email: userRequest.Email,
				Age:   userRequest.Age,
			}

			// Send user update to the server
			if err := stream.Send(req); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		// Close the send stream after sending all requests
		if err := stream.CloseSend(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		done <- true
	}()

	// Goroutine for receiving data
	go func() {
		for {
			// Receive updated user data from the server
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Append the received data to the updUsers slice
			updUsers = append(updUsers, pb.User{
				Id:    res.User.GetId(),
				Name:  res.User.GetName(),
				Email: res.User.GetEmail(),
				Age:   res.User.GetAge(),
			})
		}
		done <- true
	}()

	// Wait for both goroutines to finish
	<-done
	<-done

	// Return the updated users
	c.JSON(http.StatusOK, updUsers)
}

func main() {
	// Set up Gin router
	r := gin.Default()

	// Set up routes
	r.POST("/users", createUser)
	r.GET("/users", getUsers)
	r.PUT("/users", updateUser)
	r.DELETE("/users", deleteUser)
	r.POST("/users/stream", createUsersStream)
	r.GET("/users/stream", getUsersStream)
	r.PUT("/users/stream", UpdateUsersStream)
	r.DELETE("/users/stream", deleteUsersStream)

	// Start Gin server
	err := r.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
