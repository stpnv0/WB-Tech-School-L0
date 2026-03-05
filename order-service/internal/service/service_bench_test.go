package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"order-service/internal/models"
	"order-service/internal/service/mocks"
)

func benchSvcLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func benchSvcOrder() *models.Order {
	return &models.Order{
		OrderUID:    "bench-uid",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Alice",
			Phone:   "+71234567890",
			Zip:     "123456",
			City:    "Moscow",
			Address: "ul. Testovaya, d. 1",
			Region:  "Region",
			Email:   "alice@test.com",
		},
		Payment: models.Payment{
			Transaction:  "bench-uid",
			Currency:     "RUB",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDT:    1637907727,
			Bank:         "Sberbank",
			DeliveryCost: 200,
			GoodsTotal:   800,
		},
		Items: []models.Item{
			{
				ChrtID:      1000001,
				TrackNumber: "TRACK123",
				Price:       500,
				Rid:         "rid-1",
				Name:        "T-Shirt",
				Sale:        10,
				Size:        "42",
				TotalPrice:  450,
				NmID:        100001,
				Brand:       "Nike",
				Status:      202,
			},
		},
		Locale:          "ru",
		CustomerID:      "cust_1",
		DeliveryService: "meest",
		Shardkey:        "9",
		SmID:            99,
		DateCreated:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OofShard:        "1",
	}
}

func BenchmarkGetOrderByUID_CacheHit(b *testing.B) {
	mockCache := &mocks.OrderCache{}
	mockRepo := &mocks.OrderRepository{}
	order := benchSvcOrder()

	mockCache.On("Get", "bench-uid").Return(order, true)

	svc := NewOrderService(mockRepo, mockCache, benchSvcLogger())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.GetOrderByUID(ctx, "bench-uid")
	}

	_ = mock.Anything // avoid unused import
}
