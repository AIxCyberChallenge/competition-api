package ratelimit

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiterStore struct {
	db         *redis.Client
	limiterKey string
	perMinute  int64
	failOpen   bool
}

type RedisLimiterConfig struct {
	RedisClient *redis.Client
	LimiterKey  string
	PerMinute   int64
	FailOpen    bool
}

func (store *RedisLimiterStore) Allow(identifier string) (bool, error) {
	// This method might let N-1 extra requests in due to race condition where N is the possible number of concurrent writers
	// This is a smaller concern than the possibility that we will lose a distributed lock

	ctx := context.Background()

	key := "competitionapi-ratelimit-" + store.limiterKey + "-" + identifier

	reqsLeftStr, err := store.db.Get(ctx, key).Result()

	if err == nil {
		reqsLeft := 0

		reqsLeft, err = strconv.Atoi(reqsLeftStr)
		if err != nil {
			return store.failOpen, err
		}

		if reqsLeft == 0 {
			return false, nil
		}
	} else {
		if err != redis.Nil {
			return store.failOpen, err
		}

		store.db.Set(ctx, key, store.perMinute, 60*time.Second)
	}

	store.db.Decr(ctx, key)

	return true, nil
}

func NewRedisLimitStore(config RedisLimiterConfig) (store *RedisLimiterStore) {
	return &RedisLimiterStore{
		perMinute:  config.PerMinute,
		db:         config.RedisClient,
		limiterKey: config.LimiterKey,
	}
}
