package configs

import (
	"github.com/caarlos0/env/v6"
)

type AppConfig struct {
	HTTPPort string `env:"http_port" envDefault:":8080"`
}

type MongoDB struct {
	URI      string `env:"MONGO_URI" envDefault:"mongodb://root:rootpassword@localhost:27017/mocktool?authSource=admin"`
	Database string `env:"database" envDefault:"mocktool"`
}
type Config struct {
	AppConfig AppConfig
	MongoDB   MongoDB
}

func LoadConfig() *Config {
	var dbConfig Config
	if err := env.Parse(&dbConfig); err != nil {
		panic(err)
	}
	return &dbConfig
}
