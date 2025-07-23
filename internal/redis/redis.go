package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func InitRedis(reddisAddress string, redisUsername string, redisPassword string) {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     reddisAddress,
		Username: redisUsername,
		Password: redisPassword,
		DB:       0,
	})
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) {
	if err := Rdb.Set(ctx, key, value, expiration); err != nil {
		log.Printf("Failed to add %s to redis: %v", key, err)
		return
	}
}
