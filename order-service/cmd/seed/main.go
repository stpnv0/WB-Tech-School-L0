package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"order-service/internal/models"
	"order-service/internal/repository"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Go-программа, которая вставляет N заказов (по умолчанию 10 000) с рандомными UID в PostgreSQL.
// Использует существующий repository.SaveOrder. Сохраняет список UID в файл /tmp/order_uids.txt
// для последующего нагрузочного тестирования.
func main() {
	n := flag.Int("n", 10000, "numbers of items to insert")
	uidsFile := flag.String("out", "/tmp/order_uids.txt", "file to write generated UIDs")
	flag.Parse()

	_ = godotenv.Load()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		envOrDefault("DB_USER", "postgres"),
		envOrDefault("DB_PASSWORD", "postgres"),
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_NAME", "orders"),
	)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer pool.Close()

	if err = pool.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	repo := repository.NewPostgresRepository(pool)

	uids := make([]string, 0, *n)
	inserted := 0
	for i := 0; i < *n; i++ {
		order := generateOrder(i)
		if err = repo.SaveOrder(ctx, order); err != nil {
			log.Printf("order %d: %v", i, err)
			continue
		}
		uids = append(uids, order.OrderUID)
		inserted++

		if inserted%1000 == 0 {
			log.Printf("inserted %d/%d", inserted, *n)
		}
	}

	log.Printf("Done: inserted %d orders", inserted)

	if err = os.WriteFile(*uidsFile, []byte(strings.Join(uids, "\n")+"\n"), 0644); err != nil {
		log.Fatalf("write uids file: %v", err)
	}
	log.Printf("UIDs written to %s", *uidsFile)
}

var (
	names    = []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank"}
	cities   = []string{"Moscow", "Saint Petersburg", "Novosibirsk", "Yekaterinburg", "Kazan", "Samara"}
	banks    = []string{"Sberbank", "Tinkoff", "Alfa", "VTB", "Raiffeisen"}
	brands   = []string{"Nike", "Adidas", "Puma", "Reebok", "New Balance", "Asics"}
	products = []string{"T-Shirt", "Sneakers", "Jacket", "Pants", "Hat", "Backpack", "Socks", "Hoodie"}
)

func generateOrder(idx int) *models.Order {
	uid := fmt.Sprintf("seed_%08d_%s", idx, randString(8))
	track := fmt.Sprintf("TRACK%s", randString(10))
	now := time.Now().Add(-time.Duration(rand.Intn(720)) * time.Hour)

	itemCount := 1 + rand.Intn(3)
	items := make([]models.Item, itemCount)
	goodsTotal := 0
	for i := 0; i < itemCount; i++ {
		price := 100 + rand.Intn(9900)
		sale := rand.Intn(50)
		total := price * (100 - sale) / 100
		goodsTotal += total
		items[i] = models.Item{
			ChrtID:      int64(1000000 + rand.Intn(9000000)),
			TrackNumber: track,
			Price:       price,
			Rid:         randString(20),
			Name:        products[rand.Intn(len(products))],
			Sale:        sale,
			Size:        fmt.Sprintf("%d", 36+rand.Intn(14)),
			TotalPrice:  total,
			NmID:        int64(100000 + rand.Intn(900000)),
			Brand:       brands[rand.Intn(len(brands))],
			Status:      202,
		}
	}
	deliveryCost := 200 + rand.Intn(1300)

	return &models.Order{
		OrderUID:    uid,
		TrackNumber: track,
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    names[rand.Intn(len(names))],
			Phone:   fmt.Sprintf("+7%010d", rand.Intn(10000000000)),
			Zip:     fmt.Sprintf("%06d", rand.Intn(999999)),
			City:    cities[rand.Intn(len(cities))],
			Address: fmt.Sprintf("ul. Testovaya, d. %d", 1+rand.Intn(200)),
			Region:  "Region",
			Email:   fmt.Sprintf("user%d@test.com", idx),
		},
		Payment: models.Payment{
			Transaction:  uid,
			RequestID:    "",
			Currency:     "RUB",
			Provider:     "wbpay",
			Amount:       goodsTotal + deliveryCost,
			PaymentDT:    now.Unix(),
			Bank:         banks[rand.Intn(len(banks))],
			DeliveryCost: deliveryCost,
			GoodsTotal:   goodsTotal,
			CustomFee:    0,
		},
		Items:             items,
		Locale:            "ru",
		InternalSignature: "",
		CustomerID:        fmt.Sprintf("cust_%d", idx),
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       now,
		OofShard:          "1",
	}
}

func randString(key int) string {
	letters := "qwertyuiopasdfghjklzxcvbnm1234567890"
	b := make([]byte, key)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
