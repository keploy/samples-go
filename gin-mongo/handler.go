package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itchyny/base58-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type URL struct {
	ID      string    `json:"id" bson:"_id"`
	Created time.Time `json:"created" bson:"created"`
	Updated time.Time `json:"updated" bson:"updated"`
	URL     string    `json:"URL" bson:"url"`
}

func validateURL(urlString string) error {
	parsedURL, err := url.ParseRequestURI(urlString)
	if err != nil {
		return err
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.New("invalid URL format")
	}
	return nil
}

func Get(ctx context.Context, col *mongo.Collection, id string) (*URL, error) {
	filter := bson.M{"_id": id}
	var u URL
	err := col.FindOne(ctx, filter).Decode(&u)
	return &u, err
}

func Upsert(ctx context.Context, col *mongo.Collection, u URL) error {
	upsert := true
	opt := &options.UpdateOptions{
		Upsert: &upsert,
	}
	filter := bson.M{"_id": u.ID}
	update := bson.D{primitive.E{Key: "$set", Value: u}}
	_, err := col.UpdateOne(ctx, filter, update, opt)
	return err
}

func getURL(c *gin.Context, col *mongo.Collection, logger *zap.Logger) {
	hash := c.Param("param")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing URL hash"})
		return
	}
	
	u, err := Get(c.Request.Context(), col, hash)
	if err != nil {
		logger.Error("Failed to find URL", zap.Error(err), zap.String("hash", hash))
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}
	c.Redirect(http.StatusSeeOther, u.URL)
}

func putURL(c *gin.Context, col *mongo.Collection) {
	var m map[string]string
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	u := m["url"]
	if u == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing URL"})
		return
	}

	if err := validateURL(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
		return
	}

	t := time.Now()
	id := GenerateShortLink(u)
	urlEntry := URL{
		ID:      id,
		Created: t,
		Updated: t,
		URL:     u,
	}

	if err := Upsert(c.Request.Context(), col, urlEntry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ts":  t.UnixNano(),
		"url": "http://localhost:8080/" + id,
	})
}

func New(host, db string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(
		fmt.Sprintf("mongodb://%s/%s?retryWrites=true&w=majority", host, db),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return mongo.Connect(ctx, clientOptions)
}

func GenerateShortLink(initialLink string) string {
	urlHashBytes := sha256Of(initialLink)
	generatedNumber := new(big.Int).SetBytes(urlHashBytes).Uint64()
	finalString := base58Encoded([]byte(fmt.Sprintf("%d", generatedNumber)))
	
	// Ensure consistent length and handle potential encoding issues
	if len(finalString) < 8 {
		finalString = finalString + "00000000"
	}
	return finalString[:8]
}

func sha256Of(input string) []byte {
	algorithm := sha256.New()
	algorithm.Write([]byte(input))
	return algorithm.Sum(nil)
}

func base58Encoded(bytes []byte) string {
	encoding := base58.BitcoinEncoding
	encoded, err := encoding.Encode(bytes)
	if err != nil {
		log.Printf("Encoding error: %v", err)
		return ""
	}
	return string(encoded)
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	client, err := New("localhost", "urlshortener")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	col := client.Database("urlshortener").Collection("urls")
	
	r := gin.Default()
	r.GET("/:param", func(c *gin.Context) {
		getURL(c, col, logger)
	})
	r.POST("/shorten", func(c *gin.Context) {
		putURL(c, col)
	})

	r.Run(":8080")
}
