package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"order-service/internal/models"
	"order-service/internal/repository"
)

type OrderRepository interface {
	SaveOrder(context.Context, *models.Order) error
	GetOrderByUID(context.Context, string) (*models.Order, error)
	GetLastNOrders(context.Context, int) ([]*models.Order, error)
}

type OrderCache interface {
	Set(*models.Order)
	Get(string) (*models.Order, bool)
	LoadBatch([]*models.Order)
}

type OrderService struct {
	db    OrderRepository
	cache OrderCache
	log   *slog.Logger
}

func NewOrderService(db OrderRepository, cache OrderCache, log *slog.Logger) *OrderService {
	return &OrderService{
		db:    db,
		cache: cache,
		log:   log,
	}
}

func (s *OrderService) ProcessNewOrder(ctx context.Context, order *models.Order) error {
	const op = "OrderService.ProcessNewOrder"
	log := s.log.With(
		slog.String("op", op),
		slog.String("order_uid", order.OrderUID),
	)

	log.Info("starting to process new order")

	if err := s.db.SaveOrder(ctx, order); err != nil {
		log.Error("failed to save order to repository", slog.Any("error", err))
		return fmt.Errorf("%s: %v", op, err)
	}

	s.cache.Set(order)
	log.Info("order processed and cached successfully")

	return nil
}

func (s *OrderService) GetOrderByUID(ctx context.Context, orderUID string) (*models.Order, error) {
	const op = "OrderService.GetOrderByUID"
	log := s.log.With(
		slog.String("op", op),
		slog.String("order_uid", orderUID),
	)

	if order, ok := s.cache.Get(orderUID); ok {
		log.Info("order found in cache (cache hit)")
		return order, nil
	}

	log.Info("order not found in cache (cache miss), fetching from repository")
	order, err := s.db.GetOrderByUID(ctx, orderUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Warn("order not found in repository")
		} else {
			log.Error("failed to get order from repository", slog.Any("error", err))
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("order found in repository, updating cache")
	if order != nil {
		s.cache.Set(order)
	}

	return order, nil
}

func (s *OrderService) PreloadCache(ctx context.Context, numOrders int) error {
	const op = "OrderService.PreloadCache"
	log := s.log.With(slog.String("op", op))

	log.Info("starting cache preloading", slog.Int("orders_to_load", numOrders))
	orders, err := s.db.GetLastNOrders(ctx, numOrders)
	if err != nil {
		log.Error("failed to get last orders from repository", slog.Any("error", err))
		return fmt.Errorf("%s: %v", op, err)
	}

	s.cache.LoadBatch(orders)
	log.Info("cache preloaded successfully", slog.Int("orders_loaded", len(orders)))

	return nil
}
