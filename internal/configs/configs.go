package configs

import (
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
type Config struct {
	AppConfig AppConfig
	MongoDB   MongoDB
	RedisConf RedisConf
}

func LoadConfig() *Config {
	var dbConfig Config
	if err := env.Parse(&dbConfig); err != nil {
		panic(err)
	}
	return &dbConfig
}
