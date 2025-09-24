package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	_ "github.com/keploy/go-sdk/v3/keploy"

	"github.com/gin-gonic/gin"
)

// --- Struct Definitions ---

type User struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

type Item struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type Product struct {
	ProductID   string   `json:"product_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type ConfigUpdate struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func main() {
	// Initialize a new Gin router with default middleware
	router := gin.Default()

	// --- Original 14 Endpoints ---

	// 1. A simple health check endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 2. An endpoint that takes a name as a URL parameter
	router.GET("/hello/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.String(http.StatusOK, "Hello, %s!", name)
	})

	// 3. An endpoint that randomly returns 0 or 1
	router.GET("/random", func(c *gin.Context) {
		rand.Seed(time.Now().UnixNano())
		randomNumber := rand.Intn(2)
		c.JSON(http.StatusOK, gin.H{"value": randomNumber})
	})

	// 4. Endpoint using a query parameter
	router.GET("/welcome", func(c *gin.Context) {
		name := c.DefaultQuery("name", "Guest")
		c.JSON(http.StatusOK, gin.H{"message": "Welcome, " + name})
	})

	// 5. POST endpoint that accepts a JSON body
	router.POST("/user", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "user": user})
	})

	// 6. Endpoint that returns a static list (array of JSON objects)
	router.GET("/items", func(c *gin.Context) {
		items := []Item{
			{ID: "item1", Name: "Laptop", Price: 1200.00},
			{ID: "item2", Name: "Mouse", Price: 25.50},
			{ID: "item3", Name: "Keyboard", Price: 75.00},
		}
		c.JSON(http.StatusOK, items)
	})

	// 7. PUT endpoint to simulate updating data
	router.PUT("/item/:id", func(c *gin.Context) {
		itemID := c.Param("id")
		var updatedItem Item
		if err := c.ShouldBindJSON(&updatedItem); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Item updated successfully", "id": itemID, "updatedData": updatedItem})
	})

	// 8. DELETE endpoint to simulate deleting data
	router.DELETE("/item/:id", func(c *gin.Context) {
		itemID := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully", "id": itemID})
	})

	// 9-14. Simple GET endpoints
	router.GET("/someone", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Someone"}) })
	router.GET("/something", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Something"}) })
	router.GET("/anyone", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Anyone"}) })
	router.GET("/noone", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "No one"}) })
	router.GET("/nobody", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Nobody"}) })
	router.GET("/everyone", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Everyone"}) })

	// --- 49 New Endpoints ---

	// Group 1: More simple GET endpoints
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	router.GET("/status", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"service": "user-api", "status": "active"}) })
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"healthy": true}) })
	router.GET("/info", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": "1.0.2", "author": "Keploy"}) })
	router.GET("/timestamp", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"current_time": time.Now().UTC()}) })
	router.GET("/anything", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Anything"}) })
	router.GET("/everything", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Everything"}) })
	router.GET("/nothing", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Nothing"}) })
	router.GET("/somewhere", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Somewhere"}) })
	router.GET("/nowhere", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Nowhere"}) })
	
	// Group 2: RESTful API for 'products'
	router.GET("/products", func(c *gin.Context) {
		products := []Product{
			{ProductID: "prod001", Name: "Eco-friendly Water Bottle", Description: "A reusable bottle.", Tags: []string{"eco", "kitchen"}},
			{ProductID: "prod002", Name: "Wireless Charger", Description: "Charges your devices.", Tags: []string{"tech", "mobile"}},
		}
		c.JSON(http.StatusOK, products)
	})
	router.POST("/products", func(c *gin.Context) {
		var newProduct Product
		if err := c.ShouldBindJSON(&newProduct); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "product created", "data": newProduct})
	})
	router.GET("/products/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"product_id": id, "name": "Sample Product", "price": 99.99})
	})
	router.PUT("/products/:id", func(c *gin.Context) {
		id := c.Param("id")
		var updatedProduct Product
		if err := c.ShouldBindJSON(&updatedProduct); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": fmt.Sprintf("product %s updated", id), "data": updatedProduct})
	})
	router.DELETE("/products/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"status": fmt.Sprintf("product %s deleted", id)})
	})
	router.PATCH("/products/:id", func(c *gin.Context) {
		id := c.Param("id")
		var update map[string]interface{}
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": fmt.Sprintf("product %s partially updated", id), "patch": update})
	})

	// Group 3: More complex routing and parameters
	router.GET("/users/:userID/posts/:postID", func(c *gin.Context) {
		userID := c.Param("userID")
		postID := c.Param("postID")
		c.JSON(http.StatusOK, gin.H{"user": userID, "post": postID, "content": "This is a sample post."})
	})
	router.GET("/search", func(c *gin.Context) {
		query := c.Query("q")
		limit := c.DefaultQuery("limit", "10")
		c.JSON(http.StatusOK, gin.H{"searching_for": query, "limit": limit})
	})
	router.GET("/files/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		c.JSON(http.StatusOK, gin.H{"requested_file": filepath})
	})
	
	// Group 4: Different response types
	router.GET("/html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<h1>This is HTML</h1>"))
	})
	router.GET("/xml", func(c *gin.Context) {
		c.XML(http.StatusOK, gin.H{"user": "john", "status": "active"})
	})
	router.GET("/redirect", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "http://google.com")
	})
	
	// Group 5: More HTTP methods
	router.PATCH("/config", func(c *gin.Context) {
		var update ConfigUpdate
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "config updated", "update": update})
	})
	router.HEAD("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK) // HEAD requests don't have a body
	})
	router.OPTIONS("/resource", func(c *gin.Context) {
		c.Header("Allow", "GET, POST, OPTIONS")
		c.Status(http.StatusOK)
	})

	// Group 6: API Versioning examples
	v1 := router.Group("/api/v1")
	{
		v1.GET("/data", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": 1, "data": "legacy data"}) })
		v1.GET("/users", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": 1, "users": []string{"alpha", "beta"}}) })
		v1.POST("/users", func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"version": 1, "status": "user created"}) })
	}
	v2 := router.Group("/api/v2")
	{
		v2.GET("/data", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": 2, "payload": "new data format"}) })
		v2.GET("/users", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": 2, "users": []map[string]string{{"name": "gamma"}, {"name": "delta"}}}) })
		v2.POST("/users", func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"version": 2, "message": "user successfully registered"}) })
	}

	// Group 7: Filler endpoints to reach 63
	router.GET("/system/logs", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"log_level": "INFO", "entries": 1024}) })
	router.GET("/system/metrics", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"cpu_usage": "15%", "memory": "256MB"}) })
	router.POST("/system/reboot", func(c *gin.Context) { c.JSON(http.StatusAccepted, gin.H{"message": "System reboot initiated"}) })
	router.GET("/proxy", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"forwarding_to": "downstream-service"}) })
	router.GET("/legacy", func(c *gin.Context) { c.JSON(http.StatusGone, gin.H{"error": "This endpoint is deprecated"}) })
	router.GET("/secure/data", func(c *gin.Context) { c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"}) })
	router.GET("/admin/panel", func(c *gin.Context) { c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"}) })
	router.GET("/long-poll", func(c *gin.Context) {
		time.Sleep(1 * time.Second) // Simulate a long-running task
		c.JSON(http.StatusOK, gin.H{"status": "task complete"})
	})
	router.GET("/anybody", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Anybody"}) })
	router.GET("/everybody", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Everybody"}) })
	router.GET("/somebody", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Somebody"}) })
	router.PUT("/user/:id/password", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("password for user %s updated", c.Param("id"))}) })
	router.GET("/user/:id/profile", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"user_id": c.Param("id"), "profile": "..."}) })
	router.POST("/events", func(c *gin.Context) { c.JSON(http.StatusAccepted, gin.H{"status": "event received"}) })
	router.GET("/session/info", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"session_id": "xyz-123", "active": true}) })
	
	// Start the HTTP server on port 8080
	router.Run(":8080")
}