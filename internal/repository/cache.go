package repository

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/namnv2496/mocktool/internal/configs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type ICache interface {
	Set(ctx context.Context, key string, value any) error
	Get(ctx context.Context, key string) (any, error)
	InvalidAllKey(ctx context.Context, key string) error
}

type Cache struct {
	conf        *configs.Config
	redisClient *redis.Client
}

func NewCache(
	conf *configs.Config,
) ICache {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisConf.Host,
		Username: conf.RedisConf.Username,
		Password: conf.RedisConf.Password,
		DB:       int(conf.RedisConf.Database),
	})
	// invalid all keys
	invalidateAllKeys(context.Background(), client)
	return &Cache{
		redisClient: client,
	}
}

func (_self Cache) Set(ctx context.Context, key string, value any) error {
	return _self.redisClient.Set(ctx, key, value, redis.KeepTTL).Err()
}

func (_self Cache) Get(ctx context.Context, key string) (any, error) {
	data, err := _self.redisClient.Get(ctx, key).Result()
	log.Println("get newsfeed from redis ", key, data, err)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get data from cache")
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
