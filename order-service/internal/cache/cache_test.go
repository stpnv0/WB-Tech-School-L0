package cache

import (
	"fmt"
	"sync"
	"testing"

	"order-service/internal/models"
)

// helper: быстро делает заказ с нужным UID
func makeOrder(uid string) *models.Order {
	return &models.Order{OrderUID: uid}
}

// Базовые сценарии

func TestNewLRUCache(t *testing.T) {
	c := NewLRUCache(3)
	if c.capacity != 3 {
		t.Fatalf("want capacity 3, got %d", c.capacity)
	}
	if c.lru.Len() != 0 || len(c.cache) != 0 {
		t.Fatal("new cache must be empty")
	}
}

func TestSetAndGet(t *testing.T) {
	c := NewLRUCache(2)

	o1 := makeOrder("1")
	c.Set(o1)

	got, ok := c.Get("1")
	if !ok || got != o1 {
		t.Fatal("cannot retrieve just-set order")
	}
}

func TestUpdateExistingOrder(t *testing.T) {
	c := NewLRUCache(2)

	o1 := makeOrder("1")
	o1.Delivery.Email = "first@mail"
	c.Set(o1)

	o1new := makeOrder("1")
	o1new.Delivery.Email = "second@mail"
	c.Set(o1new) // обновляем

	got, _ := c.Get("1")
	if got.Delivery.Email != "second@mail" {
		t.Fatal("order was not updated")
	}
	if c.lru.Len() != 1 {
		t.Fatal("cache length must remain 1 after update")
	}
}

func TestEviction(t *testing.T) {
	c := NewLRUCache(2)

	c.Set(makeOrder("1"))
	c.Set(makeOrder("2"))
	c.Set(makeOrder("3")) // должен вытеснить "1"

	if _, ok := c.Get("1"); ok {
		t.Fatal("oldest element was not evicted")
	}
	if _, ok := c.Get("2"); !ok {
		t.Fatal("element 2 must still exist")
	}
	if _, ok := c.Get("3"); !ok {
		t.Fatal("element 3 must exist")
	}
}

func TestGetMovesToFront(t *testing.T) {
	c := NewLRUCache(2)

	c.Set(makeOrder("1"))
	c.Set(makeOrder("2"))
	// теперь порядок [2,1]

	// обращаемся к 1 – он должен стать первым
	c.Get("1")
	c.Set(makeOrder("3")) // должен вытеснить 2

	if _, ok := c.Get("2"); ok {
		t.Fatal("element 2 must be evicted")
	}
	if _, ok := c.Get("1"); !ok {
		t.Fatal("element 1 must still exist (was touched)")
	}
}

// LoadBatch

func TestLoadBatchEmpty(t *testing.T) {
	c := NewLRUCache(3)
	c.Set(makeOrder("A"))
	c.LoadBatch(nil) // должно очистить
	if c.lru.Len() != 0 || len(c.cache) != 0 {
		t.Fatal("LoadBatch(nil) must clear cache")
	}
}

func TestLoadBatchRespectsCapacity(t *testing.T) {
	c := NewLRUCache(2)
	orders := []*models.Order{
		makeOrder("A"),
		makeOrder("B"),
		makeOrder("C"), // лишний
	}
	c.LoadBatch(orders)
	if c.lru.Len() != 2 {
		t.Fatal("capacity must be respected")
	}
	if _, ok := c.Get("C"); ok {
		t.Fatal("third element must not be loaded")
	}
}

func TestLoadBatchReplaceOldData(t *testing.T) {
	c := NewLRUCache(2)
	c.Set(makeOrder("OLD"))
	c.LoadBatch([]*models.Order{makeOrder("NEW")})

	if _, ok := c.Get("OLD"); ok {
		t.Fatal("old data must be replaced")
	}
	if _, ok := c.Get("NEW"); !ok {
		t.Fatal("new data must be present")
	}
}

// Конкурентный доступ
func TestConcurrentAccess(t *testing.T) {
	c := NewLRUCache(100)
	var wg sync.WaitGroup
	// 1000 горутин одновременно пишут и читают
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%150) // > capacity
			c.Set(makeOrder(key))
			c.Get(key)
		}(i)
	}
	wg.Wait()
	// просто проверим, что не упало и capacity не превышена
	if c.lru.Len() != c.capacity {
		t.Fatalf("expected len %d, got %d", c.capacity, c.lru.Len())
	}
}
