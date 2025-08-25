package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"order-service/internal/cache"
	"order-service/internal/config"
	"order-service/internal/handlers"
	"order-service/internal/kafka"
	"order-service/internal/repository"
	"order-service/internal/router"
	"order-service/internal/service"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	cfg := config.MustLoad()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbPool, err := initDB(cfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer dbPool.Close()

	orderRepo := repository.NewPostgresRepository(dbPool)
	orderCache := cache.NewLRUCache(cfg.Cache.CacheCapacity)
	orderService := service.NewOrderService(orderRepo, orderCache, logger)

	ctx := context.Background()
	logger.Info("Preloading cache", slog.Int("limit", cfg.Cache.CachePreloadLimit))
	if err = orderService.PreloadCache(ctx, cfg.Cache.CachePreloadLimit); err != nil {
		logger.Error("Failed to preload cache", slog.Any("error", err))
	}

	kafkaConsumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
		cfg.Kafka.MinBytes,
		cfg.Kafka.MaxBytes,
		cfg.Kafka.MaxWait,
		logger,
		orderService,
	)

	handler := handlers.NewHandler(orderService, logger)
	r := router.InitRouter(handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Info("Starting Kafka consumer")
		kafkaConsumer.Start(ctx)
	}()

	go func() {
		logger.Info("Starting HTTP server", slog.String("address", cfg.HTTPServer.Address))
		if err := r.Run(cfg.HTTPServer.Address); err != nil {
			logger.Error("HTTP server error", slog.Any("error", err))
			cancel()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
	cancel()
}

func initDB(cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.DBName,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("Connected to PostgreSQL")
	return pool, nil
}
