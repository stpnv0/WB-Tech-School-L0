package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"order-service/internal/models"
	"order-service/internal/repository"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		panic(err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		panic(err)
	}

	// Create tables
	createTables := `
   CREATE TABLE orders (
       order_uid VARCHAR PRIMARY KEY,
       track_number VARCHAR,
       entry VARCHAR,
       locale VARCHAR,
       internal_signature VARCHAR,
       customer_id VARCHAR,
       delivery_service VARCHAR,
       shardkey VARCHAR,
       sm_id INT,
       date_created TIMESTAMP,
       oof_shard VARCHAR
   );


   CREATE TABLE payments (
       order_id VARCHAR PRIMARY KEY REFERENCES orders(order_uid),
       transaction VARCHAR,
       request_id VARCHAR,
       currency VARCHAR,
       provider VARCHAR,
       amount INT,
       payment_dt BIGINT,
       bank VARCHAR,
       delivery_cost INT,
       goods_total INT,
       custom_fee INT
   );


   CREATE TABLE delivery (
       order_id VARCHAR PRIMARY KEY REFERENCES orders(order_uid),
       name VARCHAR,
       phone VARCHAR,
       zip VARCHAR,
       city VARCHAR,
       address VARCHAR,
       region VARCHAR,
       email VARCHAR
   );


   CREATE TABLE items (
       id SERIAL PRIMARY KEY,
       order_id VARCHAR REFERENCES orders(order_uid),
       chrt_id INT,
       track_number VARCHAR,
       price INT,
       rid VARCHAR,
       name VARCHAR,
       sale INT,
       size VARCHAR,
       total_price INT,
       nm_id INT,
       brand VARCHAR,
       status INT
   );`

	_, err = testPool.Exec(ctx, createTables)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	_ = pgContainer.Terminate(ctx)
	testPool.Close()

	os.Exit(code)
}

func cleanupDB(ctx context.Context, t *testing.T) {
	_, err := testPool.Exec(ctx, "TRUNCATE TABLE items, payments, delivery, orders RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func createSampleOrder(uid string, date time.Time) *models.Order {
	return &models.Order{
		OrderUID:          uid,
		TrackNumber:       "track-" + uid,
		Entry:             "entry",
		Locale:            "en",
		InternalSignature: "sig",
		CustomerID:        "cust1",
		DeliveryService:   "service",
		Shardkey:          "shard",
		SmID:              1,
		DateCreated:       date.UTC(),
		OofShard:          "oof",
		Delivery: models.Delivery{
			Name:    "John Doe",
			Phone:   "+123456789",
			Zip:     "12345",
			City:    "City",
			Address: "Address",
			Region:  "Region",
			Email:   "email@example.com",
		},
		Payment: models.Payment{
			Transaction:  "trans",
			RequestID:    "req",
			Currency:     "USD",
			Provider:     "provider",
			Amount:       100,
			PaymentDT:    123456789,
			Bank:         "bank",
			DeliveryCost: 10,
			GoodsTotal:   90,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      1,
				TrackNumber: "item-track",
				Price:       50,
				Rid:         "rid",
				Name:        "item1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  50,
				NmID:        123,
				Brand:       "brand",
				Status:      200,
			},
		},
	}
}

func TestPostgresRepository_SaveOrder(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewPostgresRepository(testPool)
	defer cleanupDB(ctx, t)

	order := createSampleOrder("order1", time.Now())

	err := repo.SaveOrder(ctx, order)
	assert.NoError(t, err)

	got, err := repo.GetOrderByUID(ctx, "order1")
	assert.NoError(t, err)
	assert.Equal(t, order, got)
}

func TestPostgresRepository_SaveOrder_Update(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewPostgresRepository(testPool)
	defer cleanupDB(ctx, t)

	order := createSampleOrder("order1", time.Now())
	err := repo.SaveOrder(ctx, order)
	assert.NoError(t, err)

	// Update
	order.TrackNumber = "updated-track"
	order.Payment.Amount = 200
	order.Items = append(order.Items, models.Item{
		ChrtID:      2,
		TrackNumber: "item-track2",
		Price:       100,
		Rid:         "rid2",
		Name:        "item2",
		Sale:        10,
		Size:        "L",
		TotalPrice:  90,
		NmID:        456,
		Brand:       "brand2",
		Status:      201,
	})

	err = repo.SaveOrder(ctx, order)
	assert.NoError(t, err)

	got, err := repo.GetOrderByUID(ctx, "order1")
	assert.NoError(t, err)
	assert.Equal(t, order, got)
	assert.Len(t, got.Items, 2) // Old item replaced? Wait, code deletes old items
}

func TestPostgresRepository_GetOrderByUID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewPostgresRepository(testPool)
	defer cleanupDB(ctx, t)

	_, err := repo.GetOrderByUID(ctx, "nonexistent")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPostgresRepository_GetLastNOrders(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewPostgresRepository(testPool)
	defer cleanupDB(ctx, t)

	now := time.Now()
	order1 := createSampleOrder("order1", now.Add(-3*time.Hour))
	order2 := createSampleOrder("order2", now.Add(-2*time.Hour))
	order3 := createSampleOrder("order3", now.Add(-1*time.Hour))

	err := repo.SaveOrder(ctx, order1)
	assert.NoError(t, err)
	err = repo.SaveOrder(ctx, order2)
	assert.NoError(t, err)
	err = repo.SaveOrder(ctx, order3)
	assert.NoError(t, err)

	got, err := repo.GetLastNOrders(ctx, 2)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "order3", got[0].OrderUID)
	assert.Equal(t, "order2", got[1].OrderUID)
}

func TestPostgresRepository_GetLastNOrders_Empty(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewPostgresRepository(testPool)
	defer cleanupDB(ctx, t)

	got, err := repo.GetLastNOrders(ctx, 5)
	assert.NoError(t, err)
	assert.Empty(t, got)
}
