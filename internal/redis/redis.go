package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var Rdb *redis.Client

func InitRedis(redisAddress string, redisUsername string, redisPassword string) {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Username: redisUsername,
		Password: redisPassword,
		DB:       0,
	})
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) {
	err := Rdb.Set(ctx, key, value, expiration).Err()
	if err != nil {
		log.Printf("Failed to add %s to redis: %v", key, err)
		return
	}
}

func GetUnmarshalledJSON(ctx context.Context, key string, data any) {
	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Str("key", key).
			Msg("Could not find value for key in Redis")
		return
	}

	err = json.Unmarshal([]byte(val), data)
	if err != nil {
		log.Error().Err(err).
			Msg("Could not unmarshal JSON from Redis")
		return
	}
}
