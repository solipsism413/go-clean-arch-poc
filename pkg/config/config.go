// Package config provides application configuration management using Viper.
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"db"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	S3       S3Config       `mapstructure:"s3"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	OTEL     OTELConfig     `mapstructure:"otel"`
	Log      LogConfig      `mapstructure:"log"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// Addr returns the Redis address.
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	Brokers         []string `mapstructure:"brokers"`
	ConsumerGroup   string   `mapstructure:"consumer_group"`
	AutoOffsetReset string   `mapstructure:"auto_offset_reset"`
}

// S3Config holds S3 configuration.
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
	UsePathStyle    bool   `mapstructure:"use_path_style"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	SecretKey            string        `mapstructure:"secret"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer               string        `mapstructure:"issuer"`
}

// OTELConfig holds OpenTelemetry configuration.
type OTELConfig struct {
	ServiceName      string `mapstructure:"service_name"`
	ServiceVersion   string `mapstructure:"service_version"`
	ExporterEndpoint string `mapstructure:"exporter_endpoint"`
	Enabled          bool   `mapstructure:"enabled"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// GRPCConfig holds gRPC configuration.
type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// Load loads configuration from environment variables and config files.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults first
	setDefaults(v)

	// Bind environment variables explicitly (maps ENV_VAR -> viper.key)
	bindEnvVars(v)

	// Enable automatic env lookup
	v.AutomaticEnv()

	// Try to read config.yaml if exists
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/taskmanager")
	_ = v.MergeInConfig() // Ignore error - config.yaml is optional

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func bindEnvVars(v *viper.Viper) {
	// Server
	v.BindEnv("server.host", "SERVER_HOST")
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	v.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")

	// Database
	v.BindEnv("db.host", "DB_HOST")
	v.BindEnv("db.port", "DB_PORT")
	v.BindEnv("db.user", "DB_USER")
	v.BindEnv("db.password", "DB_PASSWORD")
	v.BindEnv("db.name", "DB_NAME")
	v.BindEnv("db.sslmode", "DB_SSLMODE")
	v.BindEnv("db.max_conns", "DB_MAX_CONNS")
	v.BindEnv("db.min_conns", "DB_MIN_CONNS")
	v.BindEnv("db.max_conn_lifetime", "DB_MAX_CONN_LIFETIME")
	v.BindEnv("db.max_conn_idle_time", "DB_MAX_CONN_IDLE_TIME")

	// Redis
	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("redis.port", "REDIS_PORT")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")
	v.BindEnv("redis.pool_size", "REDIS_POOL_SIZE")
	v.BindEnv("redis.min_idle_conns", "REDIS_MIN_IDLE_CONNS")

	// Kafka
	v.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	v.BindEnv("kafka.consumer_group", "KAFKA_CONSUMER_GROUP")
	v.BindEnv("kafka.auto_offset_reset", "KAFKA_AUTO_OFFSET_RESET")

	// S3
	v.BindEnv("s3.endpoint", "S3_ENDPOINT")
	v.BindEnv("s3.region", "S3_REGION")
	v.BindEnv("s3.access_key_id", "S3_ACCESS_KEY_ID")
	v.BindEnv("s3.secret_access_key", "S3_SECRET_ACCESS_KEY")
	v.BindEnv("s3.bucket", "S3_BUCKET")
	v.BindEnv("s3.use_path_style", "S3_USE_PATH_STYLE")

	// JWT
	v.BindEnv("jwt.secret", "JWT_SECRET")
	v.BindEnv("jwt.access_token_ttl", "JWT_ACCESS_TOKEN_TTL")
	v.BindEnv("jwt.refresh_token_ttl", "JWT_REFRESH_TOKEN_TTL")
	v.BindEnv("jwt.issuer", "JWT_ISSUER")

	// OTEL
	v.BindEnv("otel.service_name", "OTEL_SERVICE_NAME")
	v.BindEnv("otel.service_version", "OTEL_SERVICE_VERSION")
	v.BindEnv("otel.exporter_endpoint", "OTEL_EXPORTER_ENDPOINT")
	v.BindEnv("otel.enabled", "OTEL_ENABLED")

	// Log
	v.BindEnv("log.level", "LOG_LEVEL")
	v.BindEnv("log.format", "LOG_FORMAT")

	// gRPC
	v.BindEnv("grpc.port", "GRPC_PORT")
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")

	// Database defaults
	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5433)
	v.SetDefault("db.user", "taskmanager")
	v.SetDefault("db.password", "taskmanager_secret")
	v.SetDefault("db.name", "taskmanager")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("db.max_conns", 25)
	v.SetDefault("db.min_conns", 5)
	v.SetDefault("db.max_conn_lifetime", "1h")
	v.SetDefault("db.max_conn_idle_time", "30m")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "redis_secret")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")

	// Kafka defaults
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.consumer_group", "taskmanager")
	v.SetDefault("kafka.auto_offset_reset", "earliest")

	// S3 defaults
	v.SetDefault("s3.endpoint", "http://localhost:9000")
	v.SetDefault("s3.region", "us-east-1")
	v.SetDefault("s3.access_key_id", "minioadmin")
	v.SetDefault("s3.secret_access_key", "minioadmin123")
	v.SetDefault("s3.bucket", "taskmanager")
	v.SetDefault("s3.use_path_style", true)

	// JWT defaults
	v.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-in-production")
	v.SetDefault("jwt.access_token_ttl", "15m")
	v.SetDefault("jwt.refresh_token_ttl", "168h")
	v.SetDefault("jwt.issuer", "taskmanager")

	// OTEL defaults
	v.SetDefault("otel.service_name", "taskmanager")
	v.SetDefault("otel.service_version", "1.0.0")
	v.SetDefault("otel.exporter_endpoint", "localhost:4317")
	v.SetDefault("otel.enabled", false)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// gRPC defaults
	v.SetDefault("grpc.port", 9090)
}
