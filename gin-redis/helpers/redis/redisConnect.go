// Package redis contains implement redis database
package redis

import (
	"log"
	"os"

	"github.com/go-redis/redis"
)

var RedisClient *redis.Client

func Session() *redis.Client {
	return RedisClient
}

func Init() {
	// Connect to the Redis server
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // Redis server address
		Password: "",               // No password for local Redis, set it if needed
		DB:       0,                // Default DB
	})
	_, err := RedisClient.Ping().Result()
	if err != nil {
		log.Printf("Error initializing Redis client: %v", err)
		os.Exit(1)
	}
}
