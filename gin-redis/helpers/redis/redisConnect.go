// Package redis contains implement redis database
package redis

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func Session() *redis.Client {
	return RedisClient
}

func Init() {
	// Connect to the Redis server
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password for local Redis, set it if needed
		DB:       0,                // Default DB
		Protocol: 3,
	})
	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("Error initializing Redis client: %v", err)
		os.Exit(1)
	}
}
