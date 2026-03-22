package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/namnv2496/mocktool/internal/configs"
)

var ErrCacheMiss = errors.New("cache miss")

const (
	// mocktool:<feature>:<scenario>:<account_id>:<path>:<method>:<hash_input>
	KeyMockAPITemplate = "mocktool:%s:%s:%s:%s:%s:%s"
	KeyTemplateAll     = "mocktool:*"
	// mocktool:<feature>:<scenario>:<account_id>
	KeyScnarioTemplateAccount = "mocktool:%s:%s:%s:*"
	// mocktool:<feature>:<scenario>
	KeyScnarioTemplate = "mocktool:%s:%s:*"
	// mocktool:<feature>
	KeyFeatureTemplate = "mocktool:%s:*"
	// mocktool:seq:<feature>:<scenario>:<account_id>:<path>:<method>:<hash_input>
	KeySequenceTemplate = "mocktool:seq:%s:%s:%s:%s:%s:%s"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type ICache interface {
	Set(ctx context.Context, key string, value any) error
	SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (any, error)
	Incr(ctx context.Context, key string) (int64, error)
	IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Del(ctx context.Context, key string) error
	InvalidAllKey(ctx context.Context, key string) error
}

type Cache struct {
	conf        *configs.Config
	redisClient *redis.Client
}

func NewCache(
	conf *configs.Config,
) (ICache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisConf.Host,
		Username: conf.RedisConf.Username,
		Password: conf.RedisConf.Password,
		DB:       int(conf.RedisConf.Database),
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	// invalid all keys
	invalidateAllKeys(context.Background(), client)
	return &Cache{
		redisClient: client,
	}, nil
}

func (_self Cache) Set(ctx context.Context, key string, value any) error {
	return _self.redisClient.Set(ctx, key, value, redis.KeepTTL).Err()
}

func (_self Cache) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	return _self.redisClient.Set(ctx, key, value, ttl).Err()
}

func (_self Cache) Get(ctx context.Context, key string) (any, error) {
	data, err := _self.redisClient.Get(ctx, key).Result()
	log.Println("get newsfeed from redis ", key, data, err)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	return data, nil
}

func invalidateAllKeys(ctx context.Context, redisClient *redis.Client) error {
	var cursor uint64
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, KeyTemplateAll, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (_self Cache) Incr(ctx context.Context, key string) (int64, error) {
	return _self.redisClient.Incr(ctx, key).Result()
}

func (_self Cache) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := _self.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set TTL only on the first increment so the key eventually expires.
	if count == 1 {
		if err := _self.redisClient.Expire(ctx, key, ttl).Err(); err != nil {
			log.Printf("failed to set TTL on sequence key %s: %v", key, err)
		}
	}
	return count, nil
}

func (_self Cache) Del(ctx context.Context, key string) error {
	return _self.redisClient.Del(ctx, key).Err()
}

func (_self Cache) InvalidAllKey(ctx context.Context, key string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := _self.redisClient.Scan(ctx, cursor, key, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := _self.redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
