// Package main is the HTTP API server for the task management service.
//
//	@title          Task Manager API
//	@version        1.0
//	@description    REST API для управления задачами внутри команд: ролевая модель, история изменений, кеширование списков задач и SQL-отчёт.
//	@host           localhost:8080
//	@BasePath       /api/v1
//
//	@securityDefinitions.apikey BearerAuth
//	@in                         header
//	@name                       Authorization
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"task/internal/auth"
	"task/internal/cache"
	"task/internal/config"
	"task/internal/db"
	"task/internal/httpapi"
	"task/internal/repository"
	"task/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	database, err := repository.Open(cfg.DBDSN)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if cfg.MigrateOnStart {
		if err := db.MigrateUp(database); err != nil {
			logger.Error("run migrations", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations applied")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		logger.Error("connect redis", "error", err)
		os.Exit(1)
	}
	defer func() { _ = rdb.Close() }()

	repo := repository.New(database)
	taskCache := cache.New(rdb, cfg.CacheTTL)
	authManager := auth.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	app := service.New(repo, taskCache, authManager, logger)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(app, cfg, logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("server listening", "addr", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
