package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"time"
)

type Config struct {
	HTTPServer HTTPServer
	Cache      CacheConfig
	Postgres   PostgresConfig
	Kafka      KafkaConfig
}

type HTTPServer struct {
	Address     string        `env:"HTTP_ADDRESS"`
	Timeout     time.Duration `env:"HTTP_TIMEOUT"`
	IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT"`
}
type CacheConfig struct {
	CacheCapacity     int `env:"CACHE_CAPACITY"`
	CachePreloadLimit int `env:"CACHE_PRELOAD_LIMIT"`
}
type PostgresConfig struct {
	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	DBName   string `env:"DB_NAME"`
}

type KafkaConfig struct {
	Brokers  []string      `env:"KAFKA_BROKERS" env-separator:","`
	Topic    string        `env:"KAFKA_TOPIC"`
	GroupID  string        `env:"KAFKA_GROUP_ID"`
	MinBytes int           `env:"KAFKA_MIN_BYTES"`
	MaxBytes int           `env:"KAFKA_MAX_BYTES"`
	MaxWait  time.Duration `env:"KAFKA_MAX_WAIT"`
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("No .env file found: %v", err)
	}

	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("failed to read config from environment: " + err.Error())
	}

	return &cfg
}
