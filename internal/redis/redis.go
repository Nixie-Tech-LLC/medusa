package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func InitRedis(redisAddr string) {
	Rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) {
	if err := Rdb.Set(ctx, key, value, expiration); err != nil {
		log.Printf("Failed to add %s to redis: %v", key, err)
		return
	}
}
