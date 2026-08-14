package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr       string
	DBDSN          string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	JWTSecret      string
	JWTTTL         time.Duration
	CacheTTL       time.Duration
	MigrateOnStart bool
	RateLimitRPS   float64
	RateLimitBurst int
	MaxBodyBytes   int64
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		DBDSN:          getenv("DB_DSN", "root:root@tcp(localhost:3306)/task?parseTime=true&multiStatements=true&charset=utf8mb4&loc=Local"),
		RedisAddr:      getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),
		RedisDB:        0,
		JWTSecret:      getenv("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:         24 * time.Hour,
		CacheTTL:       5 * time.Minute,
		MigrateOnStart: getenv("MIGRATE_ON_START", "true") == "true",
		RateLimitRPS:   10,
		RateLimitBurst: 30,
		MaxBodyBytes:   1 << 20,
	}

	var err error
	if v := os.Getenv("REDIS_DB"); v != "" {
		if cfg.RedisDB, err = strconv.Atoi(v); err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
	}
	if v := os.Getenv("JWT_TTL"); v != "" {
		if cfg.JWTTTL, err = time.ParseDuration(v); err != nil {
			return nil, fmt.Errorf("invalid JWT_TTL: %w", err)
		}
	}
	if v := os.Getenv("CACHE_TTL"); v != "" {
		if cfg.CacheTTL, err = time.ParseDuration(v); err != nil {
			return nil, fmt.Errorf("invalid CACHE_TTL: %w", err)
		}
	}
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		f, perr := strconv.ParseFloat(v, 64)
		if perr != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_RPS: %w", perr)
		}
		cfg.RateLimitRPS = f
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if cfg.RateLimitBurst, err = strconv.Atoi(v); err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_BURST: %w", err)
		}
	}
	return cfg, nil
}
