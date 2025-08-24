package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"order-service/internal/models"
	"order-service/internal/repository"

	"github.com/gin-gonic/gin"
)

type OrderService interface {
	ProcessNewOrder(ctx context.Context, order *models.Order) error
	GetOrderByUID(ctx context.Context, orderUID string) (*models.Order, error)
	PreloadCache(context.Context, int) error
}

type Handler struct {
	service OrderService
	log     *slog.Logger
}

func NewHandler(service OrderService, log *slog.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

// GetOrderByUID - обработчик для GET /order/:order_uid
func (h *Handler) GetOrderByUID(c *gin.Context) {
	const op = "handler.GetOrderByUID"

	orderUID := c.Param("order_uid")

	if orderUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uid is required"})
		return
	}

	log := h.log.With(
		slog.String("op", op),
		slog.String("order_uid", orderUID),
	)

	order, err := h.service.GetOrderByUID(c.Request.Context(), orderUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.With("order not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		log.Error("failed to get order", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, order)
}
