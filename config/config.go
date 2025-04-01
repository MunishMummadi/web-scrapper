package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
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

// Load loads configuration from config file, environment variables, and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load() // Ignore error if .env file doesn't exist
	
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
	
	// Setup environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Special handling for proxy URLs as a comma-separated list
	if proxyList := os.Getenv("PROXY_URLS"); proxyList != "" {
		cfg.Proxies.Enabled = true
		cfg.Proxies.URLs = strings.Split(proxyList, ",")
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("api.host", "0.0.0.0")
	v.SetDefault("api.port", 8080)
	v.SetDefault("api.readTimeout", 30*time.Second)
	v.SetDefault("api.writeTimeout", 30*time.Second)
	v.SetDefault("api.shutdownTimeout", 10*time.Second)

	v.SetDefault("crawler.userAgent", "Scraper/1.0")
	v.SetDefault("crawler.respectRobots", true)
	v.SetDefault("crawler.defaultDelay", 1*time.Second)
	v.SetDefault("crawler.maxRetries", 3)
	v.SetDefault("crawler.retryDelay", 5*time.Second)
	v.SetDefault("crawler.workerCount", 10)
	v.SetDefault("crawler.requestTimeout", 30*time.Second)
	v.SetDefault("crawler.maxConcurrentHosts", 2)
	v.SetDefault("crawler.circuitBreakerRatio", 0.5)
	v.SetDefault("crawler.circuitBreakerTime", 5*time.Minute)
	v.SetDefault("crawler.headlessBrowser", false)
	v.SetDefault("crawler.cacheExpiration", 24*time.Hour)

	v.SetDefault("database.filepath", "./data/scraper.db")

	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("proxies.enabled", false)
	v.SetDefault("proxies.urls", []string{})
	v.SetDefault("proxies.apiKey", "")
	v.SetDefault("proxies.apiUrl", "")
}

func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *APIConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
