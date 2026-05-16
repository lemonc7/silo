package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	TMDB     TMDBConfig     `yaml:"tmdb"`
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	TZ   string `yaml:"tz" env:"TZ" env-default:"Asia/Shanghai"`
}

type TMDBConfig struct {
	BearerToken string            `yaml:"bearer_token" env:"TMDB_BEARER_TOKEN"`
	AccountID   string            `yaml:"account_id" env:"TMDB_ACCOUNT_ID"`
	Proxy       string            `yaml:"proxy" env:"TMDB_PROXY"`
	Hosts       map[string]string `yaml:"hosts"`
}

type AppConfig struct {
	URL      string `yaml:"url" env:"APP_URL"`
	Username string `yaml:"username" env:"APP_USERNAME"`
	Password string `yaml:"password" env:"APP_PASSWORD"`
}

type DatabaseConfig struct {
	// 允许的最大并发连接数（WAL 模式下多读）
	MaxOpenConns int `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	// 闲置连接数，频繁请求时避免重复创建连接
	MaxIdleConns int `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" env-default:"5"`
	// 连接的最大生命周期，防止长连接导致内存碎片
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" env-default:"1h"`

	// 锁等待超时时间（毫秒）
	BusyTimeout int `yaml:"busy_timeout" env:"DB_BUSY_TIMEOUT" env-default:"5000"`
	// 同步模式：0(OFF), 1(NORMAL), 2(FULL)。WAL 模式下设为 1(NORMAL) 性能极高且绝对安全
	Synchronous int `yaml:"synchronous" env:"DB_SYNCHRONOUS" env-default:"1"`
	// 缓存大小（页数），负数表示 KB。-8000 表示分配约 8MB 内存作为热数据缓存
	CacheSize int `yaml:"cache_size" env:"DB_CACHE_SIZE" env-default:"-8000"`
}

func (d *DatabaseConfig) DSN(path string) string {
	return fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)&_pragma=synchronous(%d)&_pragma=cache_size(%d)",
		path, d.BusyTimeout, d.Synchronous, d.CacheSize)
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
