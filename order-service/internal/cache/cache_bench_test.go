package cache

import (
	"fmt"
	"order-service/internal/models"
	"testing"
	"time"
)

func newTestOrder(uid string) *models.Order {
	return &models.Order{
		OrderUID:    uid,
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
		DateCreated:     time.Now(),
		OofShard:        "1",
	}
}

func BenchmarkLRUCache_Get_Hit(b *testing.B) {
	c := NewLRUCache(1000)
	order := newTestOrder("bench-hit")
	c.Set(order)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("bench-hit")
	}
}

func BenchmarkLRUCache_Get_Miss(b *testing.B) {
	c := NewLRUCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("nonexistent")
	}
}

func BenchmarkLRUCache_Set_New(b *testing.B) {
	c := NewLRUCache(b.N + 1)
	orders := make([]*models.Order, b.N)
	for i := range orders {
		orders[i] = newTestOrder(fmt.Sprintf("order-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(orders[i])
	}
}

func BenchmarkLRUCache_Set_Evict(b *testing.B) {
	c := NewLRUCache(100)
	// prefill
	for i := 0; i < 100; i++ {
		c.Set(newTestOrder(fmt.Sprintf("pre-%d", i)))
	}
	orders := make([]*models.Order, b.N)
	for i := range orders {
		orders[i] = newTestOrder(fmt.Sprintf("evict-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(orders[i])
	}
}

func BenchmarkLRUCache_Get_Parallel(b *testing.B) {
	c := NewLRUCache(1000)
	for i := 0; i < 100; i++ {
		c.Set(newTestOrder(fmt.Sprintf("p-%d", i)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(fmt.Sprintf("p-%d", i%100))
			i++
		}
	})
}
