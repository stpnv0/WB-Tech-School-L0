package cache

import (
	"container/list"
	"order-service/internal/models"
	"sync"
)

type LRUCache struct {
	mu       sync.Mutex
	capacity int
	cache    map[string]*list.Element
	lru      *list.List
}

type cacheItem struct {
	key   string
	order *models.Order
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Set - устанавливает заказ в LRUCache
func (c *LRUCache) Set(order *models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	//если элемент есть - переносим в голову и обновляем значение
	if elem, exists := c.cache[order.OrderUID]; exists {
		c.lru.MoveToFront(elem)
		elem.Value.(*cacheItem).order = order
		return
	}

	//если достигли лимита - удаляем самый старый элемент из списка и из мапы
	if c.capacity <= c.lru.Len() {
		lastItem := c.lru.Back()
		c.lru.Remove(lastItem)
		delete(c.cache, lastItem.Value.(*cacheItem).key)
	}

	//добавляем новый элемент в начало списка и в мапу
	newItem := &cacheItem{
		key:   order.OrderUID,
		order: order,
	}
	elem := c.lru.PushFront(newItem)
	c.cache[order.OrderUID] = elem
}

// Get - получает заказ из LRUCache по orderUID
func (c *LRUCache) Get(orderUID string) (*models.Order, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.cache[orderUID]
	if !exists {
		return nil, false
	}

	c.lru.MoveToFront(elem)
	return elem.Value.(*cacheItem).order, true
}

// LoadBatch - метод LRUCache, позволяющий предзагрузить данные на старте
func (c *LRUCache) LoadBatch(orders []*models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	//очищаем на всякий случай старый кеш
	c.cache = make(map[string]*list.Element)
	c.lru = list.New()

	// Загружаем новые данные (до capacity)
	for _, order := range orders {
		if c.lru.Len() >= c.capacity {
			break
		}
		item := &cacheItem{
			key:   order.OrderUID,
			order: order,
		}
		elem := c.lru.PushFront(item)
		c.cache[order.OrderUID] = elem
	}

}
