package ratelimiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const SlidingWindowScript = `
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])
	local limit = tonumber(ARGV[3])

	-- remove expired entries
	redis.call("ZREMRANGEBYSCORE", key, 0, now - window)

	-- count current requests
	local count = redis.call("ZCARD", key)

	if count >= limit then
		return 0
	end

	-- add new request
	redis.call("ZADD", key, now, now)
	redis.call("EXPIRE", key, math.ceil(window / 1000))

	return 1
`

type ILimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type Limiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func NewLimiter(
	client *redis.Client,
	limit int,
	window time.Duration,
) ILimiter {
	return &Limiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixMilli()

	res, err := l.client.Eval(
		ctx,
		SlidingWindowScript,
		[]string{key},
		now,
		l.window.Milliseconds(),
		l.limit,
	).Int()

	if err != nil {
		return false, err
	}

	return res == 1, nil
}
