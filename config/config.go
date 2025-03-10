package config

import (
	"time"
)

type Config struct {
	API      APIConfig
	Crawler  CrawlerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Proxies  ProxyConfig
}

type APIConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type CrawlerConfig struct {
	UserAgent           string
	RespectRobots       bool
	DefaultDelay        time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
	WorkerCount         int
	RequestTimeout      time.Duration
	MaxConcurrentHosts  int
	CircuitBreakerRatio float64
	CircuitBreakerTime  time.Duration
	HeadlessBrowser     bool
	CacheExpiration     time.Duration
}

type DatabaseConfig struct {
	FilePath string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type ProxyConfig struct {
	Enabled bool
	URLs    []string
	APIKey  string
	APIUrl  string
}

func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

}
