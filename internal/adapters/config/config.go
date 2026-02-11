package config

import (
	"time"

	"github.com/joho/godotenv"
)

type MongoConfig struct {
	URI                    string
	Database               string
	Timeout                time.Duration
	MaxPoolSize            uint64
	MinPoolSize            uint64
	ConnectTimeout         time.Duration
	ServerSelectionTimeout time.Duration
}

type RabbitMQConfig struct {
	URL             string
	MaxRetries      int
	RetryDelay      time.Duration
	ExchangeConfigs []ExchangeConfig
}

type ExchangeConfig struct {
	Name       string
	Type       string // direct, topic, fanout, headers
	Durable    bool
	AutoDelete bool
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type OutboxConfig struct {
	BatchSize int
	Interval  time.Duration
}

type HTTPConfig struct {
	Port          string
	BindInterface string
}

type Config struct {
	Mongo    MongoConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Outbox   OutboxConfig
	HTTP     HTTPConfig
	Logger   LoggerConfig
}

type LoggerConfig struct {
	Endpoint     string
	ServiceName  string
	IsProduction bool
}

func NewConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		Mongo: MongoConfig{
			URI:                    getStringEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database:               getStringEnv("MONGO_DATABASE", "challenge"),
			Timeout:                time.Duration(getIntEnv("MONGO_TIMEOUT", 10)) * time.Second,
			MaxPoolSize:            uint64(getIntEnv("MONGO_MAX_POOL_SIZE", 100)),
			MinPoolSize:            uint64(getIntEnv("MONGO_MIN_POOL_SIZE", 10)),
			ConnectTimeout:         time.Duration(getIntEnv("MONGO_CONNECT_TIMEOUT", 10)) * time.Second,
			ServerSelectionTimeout: time.Duration(getIntEnv("MONGO_SERVER_SELECTION_TIMEOUT", 5)) * time.Second,
		},
		Redis: RedisConfig{
			URL:      getStringEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getStringEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Outbox: OutboxConfig{
			BatchSize: getIntEnv("OUTBOX_BATCH_SIZE", 100),
			Interval:  time.Duration(getIntEnv("OUTBOX_INTERVAL", 500)) * time.Millisecond,
		},
		HTTP: HTTPConfig{
			Port:          getStringEnv("HTTP_PORT", "8080"),
			BindInterface: getStringEnv("HTTP_BIND_INTERFACE", "0.0.0.0"),
		},
		RabbitMQ: RabbitMQConfig{
			URL:        getStringEnv("RABBITMQ_URL", "amqp://localhost:5672"),
			MaxRetries: getIntEnv("RABBITMQ_MAX_RETRIES", 3),
			RetryDelay: time.Duration(getIntEnv("RABBITMQ_RETRY_DELAY", 1)) * time.Second,
			ExchangeConfigs: []ExchangeConfig{
				{
					Name:       getStringEnv("RABBITMQ_EXCHANGE_NAME", "exchange.order"),
					Type:       getStringEnv("RABBITMQ_EXCHANGE_TYPE", "direct"),
					Durable:    getBoolEnv("RABBITMQ_EXCHANGE_DURABLE", true),
					AutoDelete: getBoolEnv("RABBITMQ_EXCHANGE_AUTO_DELETE", false),
				},
			},
		},
		Logger: LoggerConfig{
			Endpoint:     getStringEnv("OTEL_ENDPOINT", "localhost:4317"),
			ServiceName:  getStringEnv("OTEL_SERVICE_NAME", "challenge"),
			IsProduction: getBoolEnv("IS_PRODUCTION", false),
		},
	}
}
