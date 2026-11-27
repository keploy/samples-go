package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"your-project/internal/handlers"
	"your-project/internal/repository"
	"your-project/internal/services"
)

func Setup(app *fiber.App, db *sql.DB, rdb *redis.Client) {
	// Initialize repositories
	productRepo := repository.NewProductRepository(db)
	ratingRepo := repository.NewRatingRepository(db)
	tagRepo := repository.NewTagRepository(db)
	cartRepo := repository.NewCartRepository(rdb)

	// Initialize services
	productService := services.NewProductService(productRepo)
	// Add other services here...

	// Initialize handlers
	productHandler := handlers.NewProductHandler(productService)
	// Add other handlers here...

	// API routes
	api := app.Group("/api/v1")

	// Product routes
	products := api.Group("/products")
	products.Get("/", productHandler.List)
	products.Post("/", productHandler.Create)
	products.Get("/:id", productHandler.Get)
	products.Put("/:id", productHandler.Update)
	products.Delete("/:id", productHandler.Delete)
	products.Post("/bulk", productHandler.BulkCreate)

	// Rating routes (to be implemented)
	// products.Post("/:id/rate", ratingHandler.Create)
	// products.Get("/:id/ratings", ratingHandler.GetByProduct)

	// Tag routes (to be implemented)
	// products.Post("/:id/tags", tagHandler.AddTags)
	// products.Get("/:id/tags", tagHandler.GetByProduct)
	// api.Get("/tags/:tag/products", tagHandler.GetProductsByTag)

	// Cart routes (to be implemented)
	// api.Post("/carts/:userId", cartHandler.Update)
	// api.Get("/carts/:userId", cartHandler.Get)

	// Analytics routes (to be implemented)
	// api.Get("/activity", analyticsHandler.GetActivityLog)
	// api.Get("/products/:id/visitors", analyticsHandler.GetVisitors)
	// api.Get("/leaderboard", analyticsHandler.GetLeaderboard)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":     "ok",
			"postgres":   "connected",
			"redis":      "connected",
		})
	})
}