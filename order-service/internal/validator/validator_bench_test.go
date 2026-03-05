package validator

import (
	"order-service/internal/models"
	"testing"
	"time"
)

func benchValidOrder() *models.Order {
	uid := "bench-uid-123"
	track := "TRACK123"
	return &models.Order{
		OrderUID:    uid,
		TrackNumber: track,
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
			Transaction:  uid,
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
				TrackNumber: track,
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
			{
				ChrtID:      1000002,
				TrackNumber: track,
				Price:       700,
				Rid:         "rid-2",
				Name:        "Sneakers",
				Sale:        5,
				Size:        "44",
				TotalPrice:  665,
				NmID:        100002,
				Brand:       "Adidas",
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

func BenchmarkValidate(b *testing.B) {
	order := benchValidOrder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(nil, order)
	}
}
