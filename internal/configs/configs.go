package configs

import (
	"time"

	"github.com/caarlos0/env/v6"
)

type AppConfig struct {
	HTTPPort string `env:"http_port" envDefault:":8081"`

	FowardHTTPPort string `env:"foward_http_port" envDefault:":8082"`
}

type MongoDB struct {
	URI      string `env:"MONGO_URI" envDefault:"mongodb://root:rootpassword@localhost:27017/mocktool?authSource=admin"`
	Database string `env:"database" envDefault:"mocktool"`
}

type RedisConf struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost:6379"`
	Username string `env:"REDIS_USERNAME" envDefault:""`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	Database int64  `env:"REDIS_DATABASE" envDefault:"0"`
}

type RateLimiterCfg struct {
	Host        string        `env:"RATELIMITER_HOST" envDefault:"localhost:6379"`
	DB          int           `env:"RATELIMITER_DB" envDefault:"7"`
	Limit       int           `env:"RATELIMITER_LIMIT" envDefault:"500"`
	LimitOption string        `env:"RATELIMITER_LIMIT_OPTION" envDefault:"account_id"`
	Window      time.Duration `env:"RATELIMITER_WINDOW" envDefault:"60s"`
}

type LoadSheddingCfg struct {
	MaxConcurrency int64 `env:"LOAD_SHEDDING_MAX_CONCURRENCY" envDefault:"500"`
	WarningLatency int64 `env:"LOAD_SHEDDING_WARNING_LATENCY" envDefault:"1000"` // 1 second
	MaxLatency     int64 `env:"LOAD_SHEDDING_MAX_LATENCY" envDefault:"2000"`     // 2 seconds
}

type Config struct {
	AppConfig       AppConfig
	MongoDB         MongoDB
	RedisConf       RedisConf
	RateLimiterCfg  RateLimiterCfg
	LoadSheddingCfg LoadSheddingCfg
}

func LoadConfig() *Config {
	var dbConfig Config
	if err := env.Parse(&dbConfig); err != nil {
		panic(err)
	}
	return &dbConfig
}
