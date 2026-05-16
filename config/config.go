package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	TMDB   TMDBConfig   `yaml:"tmdb"`
	App    AppConfig    `yaml:"app"`
}

type ServerConfig struct {
	Port         int           `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" env-default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" env-default:"10s"`
	TZ           string        `yaml:"tz" env:"TZ" env-default:"Asia/Shanghai"`
}

type TMDBConfig struct {
	BearerToken string `yaml:"bearer_token" env:"TMDB_BEARER_TOKEN"`
	AccountID   string `yaml:"account_id" env:"TMDB_ACCOUNT_ID"`
}

type AppConfig struct {
	URL      string `yaml:"url" env:"APP_URL"`
	Username string `yaml:"username" env:"APP_USERNAME"`
	Password string `yaml:"password" env:"APP_PASSWORD"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
