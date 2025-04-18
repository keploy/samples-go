package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Product struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name" binding:"required"`
	Price      float64                `json:"price" binding:"required"`
	Quantity   int                    `json:"quantity" binding:"required"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	RelatedIDs []string               `json:"related_ids,omitempty"`
	Categories []string               `json:"categories,omitempty"`
}

var ctx = context.Background()
var rdb *redis.Client

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	router := gin.Default()

	// Product routes
	router.GET("/products", listProducts)
	router.POST("/products", createProduct)
	router.GET("/products/:id", getProduct)
	router.PUT("/products/:id", updateProduct)
	router.DELETE("/products/:id", deleteProduct)

	// Complex data structure routes
	router.POST("/products/:id/rate", rateProduct)
	router.GET("/products/:id/ratings", getProductRatings)
	router.POST("/products/:id/tags", addProductTags)
	router.GET("/products/:id/tags", getProductTags)
	router.GET("/tags/:tag/products", getProductsByTag)
	router.POST("/products/bulk", bulkCreateProducts)
	router.GET("/activity", getActivityLog)
	router.GET("/products/:id/visitors", getProductVisitors)
	router.GET("/leaderboard", getLeaderboard)
	router.POST("/carts/:userId", updateCart)
	router.GET("/carts/:userId", getCart)

	router.GET("/health", healthCheck)

	log.Println("Server running on :8080")
	router.Run(":8080")
}

// Handlers for complex data structures
func rateProduct(c *gin.Context) {
	id := c.Param("id")
	var rating struct {
		Score float64 `json:"score" binding:"required,min=1,max=5"`
	}
	if err := c.ShouldBindJSON(&rating); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store individual ratings in a sorted set
	rdb.ZAdd(ctx, "product:"+id+":ratings", redis.Z{
		Score:  rating.Score,
		Member: uuid.New().String(),
	})

	// Update rating summary in a hash
	rdb.HIncrBy(ctx, "product:"+id+":rating_summary", "total_ratings", 1)
	rdb.HIncrByFloat(ctx, "product:"+id+":rating_summary", "total_score", rating.Score)

	c.JSON(http.StatusOK, gin.H{"message": "Rating added"})
}

func getProductRatings(c *gin.Context) {
	id := c.Param("id")

	// Get rating summary from hash
	summary, err := rdb.HGetAll(ctx, "product:"+id+":rating_summary").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get top ratings from sorted set
	ratings, err := rdb.ZRevRangeWithScores(ctx, "product:"+id+":ratings", 0, 4).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, _ := strconv.ParseFloat(summary["total_ratings"], 64)
	score, _ := strconv.ParseFloat(summary["total_score"], 64)

	response := gin.H{
		"average":       score / total,
		"total_ratings": total,
		"top_ratings":   ratings,
	}

	c.JSON(http.StatusOK, response)
}

func addProductTags(c *gin.Context) {
	id := c.Param("id")
	var tags struct {
		Tags []string `json:"tags" binding:"required"`
	}
	if err := c.ShouldBindJSON(&tags); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add tags to product's set
	pipe := rdb.TxPipeline()
	for _, tag := range tags.Tags {
		pipe.SAdd(ctx, "product:"+id+":tags", tag)
		pipe.SAdd(ctx, "tag:"+tag+":products", id)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tags added"})
}

func getProductTags(c *gin.Context) {
	id := c.Param("id")
	tags, err := rdb.SMembers(ctx, "product:"+id+":tags").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func getProductsByTag(c *gin.Context) {
	tag := c.Param("tag")
	productIDs, err := rdb.SMembers(ctx, "tag:"+tag+":products").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	products := make([]Product, 0)
	for _, id := range productIDs {
		p, err := getProductFromRedis("product:" + id)
		if err == nil {
			products = append(products, *p)
		}
	}

	c.JSON(http.StatusOK, products)
}

func bulkCreateProducts(c *gin.Context) {
	var products []Product
	if err := c.ShouldBindJSON(&products); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pipe := rdb.Pipeline()
	for i := range products {
		products[i].ID = uuid.New().String()
		err := saveProductToRedis(pipe, &products[i])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"created": len(products)})
}

func getActivityLog(c *gin.Context) {
	// Get last 10 activities from list
	activities, err := rdb.LRange(ctx, "activity_log", 0, 9).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, activities)
}

func getProductVisitors(c *gin.Context) {
	id := c.Param("id")
	// Get unique visitor count using HyperLogLog
	count, err := rdb.PFCount(ctx, "product:"+id+":visitors").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unique_visitors": count})
}

func getLeaderboard(c *gin.Context) {
	// Get top 10 products by sales using sorted set
	results, err := rdb.ZRevRangeWithScores(ctx, "product_leaderboard", 0, 9).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	leaderboard := make([]gin.H, 0)
	for _, res := range results {
		id := res.Member.(string)
		p, err := getProductFromRedis("product:" + id)
		if err == nil {
			leaderboard = append(leaderboard, gin.H{
				"product": p,
				"sales":   res.Score,
			})
		}
	}

	c.JSON(http.StatusOK, leaderboard)
}

func updateCart(c *gin.Context) {
	userID := c.Param("userId")
	var cart map[string]int
	if err := c.ShouldBindJSON(&cart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store cart as Redis hash
	pipe := rdb.Pipeline()
	pipe.Del(ctx, "cart:"+userID)
	for productID, quantity := range cart {
		pipe.HSet(ctx, "cart:"+userID, productID, quantity)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cart updated"})
}

func getCart(c *gin.Context) {
	userID := c.Param("userId")
	cart, err := rdb.HGetAll(ctx, "cart:"+userID).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make(map[string]int)
	for k, v := range cart {
		quantity, _ := strconv.Atoi(v)
		response[k] = quantity
	}

	c.JSON(http.StatusOK, response)
}

// Modified helpers to support Redis data structures
func saveProductToRedis(pipe redis.Pipeliner, p *Product) error {
	metadataJSON, _ := json.Marshal(p.Metadata)
	relatedJSON, _ := json.Marshal(p.RelatedIDs)
	categoriesJSON, _ := json.Marshal(p.Categories)

	err := pipe.HSet(ctx, "product:"+p.ID,
		"id", p.ID,
		"name", p.Name,
		"price", p.Price,
		"quantity", p.Quantity,
		"metadata", metadataJSON,
		"related_ids", relatedJSON,
		"categories", categoriesJSON,
	).Err()

	// Add to leaderboard sorted set
	pipe.ZAdd(ctx, "product_leaderboard", redis.Z{Score: 0, Member: p.ID})

	// Log activity
	pipe.LPush(ctx, "activity_log", fmt.Sprintf("Product %s created at %v", p.ID, time.Now()))

	return err
}

func getProductFromRedis(key string) (*Product, error) {
	data, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	p := &Product{
		ID:       data["id"],
		Name:     data["name"],
		Price:    parseFloat(data["price"]),
		Quantity: parseInt(data["quantity"]),
	}

	json.Unmarshal([]byte(data["metadata"]), &p.Metadata)
	json.Unmarshal([]byte(data["related_ids"]), &p.RelatedIDs)
	json.Unmarshal([]byte(data["categories"]), &p.Categories)

	return p, nil
}

// Helper functions
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// Existing handlers (listProducts, createProduct, etc.) need to be modified
// to use the updated saveProductToRedis and getProductFromRedis functions
// Handlers
func listProducts(c *gin.Context) {
	keys, err := rdb.Keys(ctx, "product:*").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	products := make([]Product, 0, len(keys))
	for _, key := range keys {
		product, err := getProductFromRedis(key)
		if err != nil {
			continue // skip invalid entries
		}
		products = append(products, *product)
	}

	c.JSON(http.StatusOK, products)
}

func createProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product.ID = uuid.New().String()
	var p redis.Pipeliner
	err := saveProductToRedis(p,&product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func getProduct(c *gin.Context) {
	id := c.Param("id")
	product, err := getProductFromRedis("product:" + id)
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func updateProduct(c *gin.Context) {
	id := c.Param("id")
	var updateData Product
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingProduct, err := getProductFromRedis("product:" + id)
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if updateData.Name != "" {
		existingProduct.Name = updateData.Name
	}
	if updateData.Price != 0 {
		existingProduct.Price = updateData.Price
	}
	if updateData.Quantity != 0 {
		existingProduct.Quantity = updateData.Quantity
	}

	var p redis.Pipeliner
	err = saveProductToRedis(p,existingProduct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existingProduct)
}

func deleteProduct(c *gin.Context) {
	id := c.Param("id")
	err := rdb.Del(ctx, "product:"+id).Err()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "redis": "connected"})
}
