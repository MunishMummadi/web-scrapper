package config

import (
	"fmt"
	"os"
	"strings"
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
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading the config file: %w", err)
		}
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w".err)
	}

	if proxyList := os.Getenv("PROXY_URLS"); proxyList != "" {
		cfg.Proxies.Enabled = true
		cfg.Proxies.URLs = strings.Split(proxyList, ",")
	}

	return &cfg, nil
}
