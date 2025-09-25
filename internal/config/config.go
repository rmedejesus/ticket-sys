package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Port         string
		Host         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
	}

	Database struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
		URL      string
	}

	JWT struct {
		Secret        string
		TokenExpiry   time.Duration
		RefreshExpiry time.Duration
	}

	Environment string
}

func Load() (*Config, error) {
	godotenv.Load() // Load .env if exists

	cfg := &Config{}

	// Server config
	cfg.Server.Port = getEnv("SERVER_PORT", "5431")
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.ReadTimeout = time.Second * 15
	cfg.Server.WriteTimeout = time.Second * 15

	// Database config
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnv("DB_PORT", "5432")
	cfg.Database.User = getEnv("DB_USER", "rmedejesus")
	cfg.Database.Password = getEnv("DB_PASSWORD", "rmdj1q2w3e")
	cfg.Database.DBName = getEnv("DB_NAME", "ticketing_db")
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", "disable")
	cfg.Database.URL = getEnv("DB_URL", "postgresql://postgres.bjokbfwbgmizvyrkonzs:UYhOthonE04P7t9G@aws-1-ap-southeast-1.pooler.supabase.com:6543/postgres")

	// JWT config
	cfg.JWT.Secret = getEnv("JWT_SECRET", "cce464308fc5156d964d4f525886e33db23e7d48e2d87a13b0b4bfb8468425e7")
	cfg.JWT.TokenExpiry = time.Minute       // 24 hours
	cfg.JWT.RefreshExpiry = time.Hour * 168 // 7 days

	cfg.Environment = getEnv("ENV", "development")

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}
