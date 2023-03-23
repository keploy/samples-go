package main

import (
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/facebookincubator/ent"
	"github.com/facebookincubator/ent/dialect/sql"
	"github.com/facebookincubator/ent/entc"
	"github.com/facebookincubator/ent/entc/gen"
)

// User defines the schema for the "user" table in the database.
type User struct {
	ent.Schema
}

// validateUser is a custom validation function for the User struct.
func validateUser(fl validator.FieldLevel) bool {
	// Validation logic goes here.
	return true
}

// createUserHandler is the HTTP handler for creating a new user.
func createUserHandler(c echo.Context) error {
	// Parse input from the request.
	var input User
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, "invalid input")
	}

	// Validate input using the "go-playground/validator" package.
	if err := validator.New().Struct(input); err != nil {
		return c.JSON(http.StatusBadRequest, "invalid input")
	}

	// Save the new user to the database using the "ent" package.
	client, err := entc.Open("mysql", "user:password@tcp(localhost:3306)/mydb")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "database error")
	}
	defer client.Close()

	user, err := client.User.Create().
		SetName(input.Name).
		SetEmail(input.Email).
		Save(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "database error")
	}

	// Return the new user as JSON.
	return c.JSON(http.StatusOK, user)
}

func main() {
	// Generate the "ent" package code based on the User struct.
	err := entc.GenerateCode(&gen.Config{
		Target: "ent/schema",
		Schema: &User{},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Set up the Echo web server with middleware.
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Set up the HTTP routes.
	e.POST("/users", createUserHandler)

	// Start the web server.
	e.Logger.Fatal(e.Start(":8080"))
}