package repository

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type CartRepository struct {
	rdb *redis.Client
}

func NewCartRepository(rdb *redis.Client) *CartRepository {
	return &CartRepository{rdb: rdb}
}

func (r *CartRepository) Update(ctx context.Context, userID string, cart map[string]int) error {
	key := "cart:" + userID
	
	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, key)
	
	for productID, quantity := range cart {
		pipe.HSet(ctx, key, productID, quantity)
	}
	
	_, err := pipe.Exec(ctx)
	return err
}

func (r *CartRepository) Get(ctx context.Context, userID string) (map[string]int, error) {
	key := "cart:" + userID
	result, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	cart := make(map[string]int)
	for productID, quantityStr := range result {
		quantity, _ := strconv.Atoi(quantityStr)
		cart[productID] = quantity
	}

	return cart, nil
}

func (r *CartRepository) LogActivity(ctx context.Context, message string) error {
	return r.rdb.LPush(ctx, "activity_log", message).Err()
}

func (r *CartRepository) GetActivityLog(ctx context.Context, limit int64) ([]string, error) {
	return r.rdb.LRange(ctx, "activity_log", 0, limit-1).Result()
}

func (r *CartRepository) IncrementVisitor(ctx context.Context, productID string, visitorID string) error {
	key := "product:" + productID + ":visitors"
	return r.rdb.PFAdd(ctx, key, visitorID).Err()
}

func (r *CartRepository) GetVisitorCount(ctx context.Context, productID string) (int64, error) {
	key := "product:" + productID + ":visitors"
	return r.rdb.PFCount(ctx, key).Result()
}

func (r *CartRepository) UpdateLeaderboard(ctx context.Context, productID string, sales float64) error {
	return r.rdb.ZAdd(ctx, "product_leaderboard", redis.Z{
		Score:  sales,
		Member: productID,
	}).Err()
}

func (r *CartRepository) GetLeaderboard(ctx context.Context, limit int64) ([]redis.Z, error) {
	return r.rdb.ZRevRangeWithScores(ctx, "product_leaderboard", 0, limit-1).Result()
}