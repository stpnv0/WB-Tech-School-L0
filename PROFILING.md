# Профилирование и оптимизация Order Processing Service


## 1. Инструменты

### 1.1 pprof

Встроенный профилировщик Go. Debug-сервер на `:6060` (`cmd/app/debug.go`) экспортирует HTTP-эндпоинты:

```go
import _ "net/http/pprof"

func startDebugServer(log *slog.Logger) {
    go http.ListenAndServe(":6060", nil)
}
```

Снятие профиля (под нагрузкой):
```bash
curl -o profiles/cpu.prof 'http://localhost:6060/debug/pprof/profile?seconds=30'
curl -o profiles/heap.prof http://localhost:6060/debug/pprof/heap
curl -o profiles/allocs.prof http://localhost:6060/debug/pprof/allocs
curl -o profiles/trace.out 'http://localhost:6060/debug/pprof/trace?seconds=5'
```

Чтение:
```bash
go tool pprof -text profiles/cpu.prof           # flat - CPU в самой функции
go tool pprof -text -cum profiles/cpu.prof       # cum - включая всё, что функция вызывает
go tool pprof -text profiles/allocs.prof         # кто сколько аллоцировал
```

### 1.2 Бенчмарки

```bash
go test -bench=. -benchmem -count=10 ./internal/cache/ | tee benchmarks/before.txt
```

Сравнение before/after через benchstat:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat benchmarks/before.txt benchmarks/after.txt
```

### 1.3 Loadtest

Утилита `cmd/loadtest/main.go` - запускает N параллельных воркеров, шлёт HTTP-запросы, собирает RPS, latency p50/p95/p99.

```bash
go run ./cmd/loadtest/ -url http://localhost:8081 -concurrency 50 -duration 35s
```

### 1.4 Trace

Запись каждого события Go runtime. Показывает mutex contention, GC паузы, scheduling delays.

```bash
go tool trace -pprof=sync profiles/trace.out > trace_sync.prof
go tool trace -pprof=sched profiles/trace.out > trace_sched.prof
go tool pprof -text trace_sync.prof
```

### 1.5 Escape analysis

Показывает, какие переменные утекают на heap (аллокация + GC).

```bash
go build -gcflags='-m -m' ./internal/validator/ 2>&1 | head -40
```

### 1.6 pprof diff

Сравнение before/after профилей:
```bash
go tool pprof -diff_base profiles/cpu.prof -text profiles/cpu_after.prof
```

---

## 2. Процесс оптимизации

### Шаг 1: Бенчмарки - фиксируем baseline

Написаны бенчмарки для каждого слоя:

| Файл                                | Что тестирует             |
|-------------------------------------|---------------------------|
| `cache/cache_bench_test.go`         | Get/Set/Parallel          |
| `service/service_bench_test.go`     | GetOrderByUID с мок-кэшем |
| `handlers/handler_bench_test.go`    | Полный HTTP цикл          |
| `validator/validator_bench_test.go` | Validate заказа           |

Результат (before):
```
Handler cache hit:  24,260 ns/op   13,580 B/op   84 allocs/op
Service cache hit:  11,450 ns/op    3,740 B/op   36 allocs/op
Validator:             528 ns/op      104 B/op    7 allocs/op
Cache Get:              24 ns/op        0 B/op    0 allocs/op
```

### Шаг 2: Loadtest - baseline

```
RPS: 762, 
Latency:
  p50: 67ms, 
  p95: 124ms, 
  p99: 228ms
```

### Шаг 3: Снять pprof под нагрузкой

Запустили loadtest на 35s, параллельно сняли CPU (30s), allocs, heap, trace (5s).

### Шаг 4: Анализ профилей - узкие места

#### Allocs (cum)

```
  cum%   функция
 83.78%  gin.LoggerWithConfig.func1             - все запросы через gin
 80.95%  handlers.(*Handler).GetOrderByUID
 61.96%  service.(*OrderService).GetOrderByUID
 54.98%  repository.GetOrderByUID               - запросы в БД
```

#### Allocs (flat - непосредственные виновники)

```
  flat     flat%   функция
 34.53MB   15%    pgconn.convertRowDescription      - pgx парсит ответ БД
 33.01MB   14%    repository.GetOrderByUID          - формирует Order из rows
 26.51MB   12%    pgx.(*baseRows).Scan
 24.03MB   10%    encoding/json.Marshal             - JSON через рефлексию
  7.00MB    3%    slog/internal/buffer.WriteString  - slog пишет строки
  6.00MB    3%    slog.(*commonHandler).clone       - slog.With() клонирование
  6.00MB    3%    slog.argsToAttrSlice
```

Выводы:
1. **repository = 55% allocs**
2. **slog = 20.5MB (9%)** - `slog.With()` клонирует logger на каждый запрос
3. **encoding/json = 24MB (10.5%)** - рефлексия на каждый Marshal
4. **gin.Logger** - 84% cum

#### EXPLAIN ANALYZE

```sql
EXPLAIN ANALYZE SELECT * FROM items WHERE order_id = 'seed_00000000_wa1e3wkq';
-- Без индекса: Parallel Seq Scan, 180000 строк, 32ms
-- С индексом:  Index Scan, 0.18ms (x177 быстрее)
```

#### Trace

- Mutex contention на LRU: 57ms - не проблема (не стоит оптимизировать)
- `semaphore.Acquire` (pgxpool): 372ms - горутины ждут соединение (default 4, а воркеров 50)

---

## 3. Оптимизации

### 3.1 `gin.Default()` -> `gin.New()` + Recovery

`gin.LoggerWithConfig` = 84% cum allocs. Пишет лог в stdout через syscall на каждый запрос.

```go
// Было:
router := gin.Default()  // gin.New() + Logger() + Recovery()

// Стало:
router := gin.New()
router.Use(gin.Recovery())
```

**Эффект**: `gin.LoggerWithConfig` = **-4.06s** cum CPU.

### 3.2 Убрать `slog.With()` и логирование cache hit

`slog.With()` = 20.5MB cum allocs (9%). На каждый вызов - clone logger + создание Attr slice

```go
// Было:
func (s *OrderService) GetOrderByUID(ctx context.Context, orderUID string) (*models.Order, error) {
    log := s.log.With(slog.String("op", op), slog.String("order_uid", orderUID))
    if order, ok := s.cache.Get(orderUID); ok {
        log.Info("cache hit")
        return order, nil
    }
    ...
}

// Стало:
func (s *OrderService) GetOrderByUID(ctx context.Context, orderUID string) (*models.Order, error) {
    if order, ok := s.cache.Get(orderUID); ok {
        return order, nil  
    }
    // логируем только error/warn
    ...
}
```

**Эффект**: Service allocs/op: **36 -> 26 (-28%)**, B/op: **-13%**.

### 3.3 Cache capacity 1000 -> X

`repository.GetOrderByUID` = 126MB (55% allocs). При capacity=1000 и 10000 заказов ~90% запросов - cache miss -> PostgreSQL
Можно изменить на разные значения для тестирования при нагрузке. 

```env
# Было:
CACHE_CAPACITY=1000
CACHE_PRELOAD_LIMIT=500

# Стало:
CACHE_CAPACITY=3000
CACHE_PRELOAD_LIMIT=3000
```

Также pre-allocated map в `LoadBatch`:
```go
c.cache = make(map[string]*list.Element, len(orders))
```

### 3.4 encoding/json -> goccy/go-json


`encoding/json.Marshal` = 24MB (10.5% allocs). Стандартный `encoding/json` использует reflect на каждый Marshal.

```go
// Было:
c.JSON(http.StatusOK, order)

// Стало:
c.Status(http.StatusOK)
c.Header("Content-Type", "application/json; charset=utf-8")
encoder := gojson.NewEncoder(c.Writer)
encoder.Encode(order)
```

**Эффект**: Handler allocs/op: **84 -> 73 (-13.10%)**, B/op: **-9.20%**.

### 3.5 Индекс на items.order_id + pgxpool MaxConns


Parallel Seq Scan 32ms -> Index Scan 0.18ms (x177). pgxpool default MaxConns=4, а воркеров 50.

```sql
CREATE INDEX IF NOT EXISTS idx_items_order_id ON items(order_id);
```

```go
type PostgresConfig struct {
    MaxConns int32 `env:"DB_MAX_CONNS" env-default:"20"`
}
```

---

## 4. Результаты

### Loadtest

Все тесты: 10000 заказов в БД, 50 конкурентных воркеров, 35s.

**При том же размере кэша (1000)** - чистый эффект оптимизаций кода и БД:

| Метрика | Before (1000) | After (1000) | Улучшение  |
|---------|---------------|--------------|------------|
| RPS     | 762           | 5 101        | **x6.7**   |
| p50     | 67.0ms        | 7.6ms        | **x8.8**   |
| p95     | 124.2ms       | 21.2ms       | **x5.9**   |
| p99     | 228.2ms       | 44.7ms       | **x5.1**   |

**При разных размерах кэша** - влияние hit rate:

| Cache size | Hit rate | RPS     | p50     | p95      | p99      |
|------------|----------|---------|---------|----------|----------|
| 1000       | ~10%     | 5 101   | 7.6ms   | 21.2ms   | 44.7ms   |
| 2000       | ~20%     | 5 691   | 7.4ms   | 19.4ms   | 31.0ms   |
| 3000       | ~30%     | 6 960   | 6.1ms   | 15.4ms   | 27.5ms   |

### Benchstat

| Бенчмарк           | sec/op       | B/op         | allocs/op              |
|--------------------|--------------|--------------|------------------------|
| Handler CacheHit   | **-8.72%**   | **-9.20%**   | **-13.10%** (84->73)   |
| Service CacheHit   | **-17.74%**  | **-12.93%**  | **-27.78%** (36->26)   |
| Validator          | ~            | ~            | ~                      |

### pprof diff

CPU (отрицательные = лучше):
- `gin.LoggerWithConfig`: **-4.06s**
- `repository.GetOrderByUID`: **-1.68s**
- `handlers.GetOrderByUID`: **-2.45s**

Allocs:
- `repository.GetOrderByUID`: **-126MB**
- `slog.(*Logger).With`: **-20.5MB**
- `encoding/json.Marshal`: **-24MB**
- `gin.LoggerWithConfig`: **-192MB** cum
