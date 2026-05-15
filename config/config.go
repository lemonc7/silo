package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config 全局配置根结构。
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	TMDB      TMDBConfig      `yaml:"tmdb"`
	App       AppConfig       `yaml:"app"`
	Download  DownloadConfig  `yaml:"download"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

// ── Server ───────────────────────────────────────

type ServerConfig struct {
	Port         int           `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" env-default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" env-default:"10s"`
}

// ── TMDB ─────────────────────────────────────────

type TMDBConfig struct {
	BearerToken string `yaml:"bearer_token" env:"TMDB_BEARER_TOKEN"`
	AccountID   string `yaml:"account_id" env:"TMDB_ACCOUNT_ID" env-default:""`
}

// ── App (资源站) ─────────────────────────────────

type AppConfig struct {
	URL      string `yaml:"url" env:"APP_URL"`
	Username string `yaml:"username" env:"APP_USERNAME"`
	Password string `yaml:"password" env:"APP_PASSWORD"`
	TZ       string `yaml:"tz" env:"TZ" env-default:"Asia/Shanghai"`
}

// ── Download ─────────────────────────────────────

type DownloadConfig struct {
	Dir         string `yaml:"dir" env:"DOWNLOAD_DIR" env-default:"./downloads"`
	Aria2URL    string `yaml:"aria2_url" env:"ARIA2_URL"`
	Aria2Secret string `yaml:"aria2_secret" env:"ARIA2_SECRET"`
}

// ── Scheduler ────────────────────────────────────

type SchedulerConfig struct {
	SyncCron string `yaml:"sync_cron" env:"SYNC_CRON" env-default:"0 */6 * * *"`
}

// ── Load ─────────────────────────────────────────

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
