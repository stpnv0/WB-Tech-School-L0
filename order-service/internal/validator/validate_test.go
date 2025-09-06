package validator_test

import (
	"errors"
	"log/slog"
	"order-service/internal/models"
	"order-service/internal/validator"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper для создания валидного заказа
func createValidOrder() *models.Order {
	return &models.Order{
		OrderUID:    "b563feb7b2b84b6test",
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  "b563feb7b2b84b6test",
			RequestID:    "req123",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDT:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
	}
}

// Helper для создания тестового логгера
func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func TestValidate_ValidOrder(t *testing.T) {
	order := createValidOrder()
	logger := createTestLogger()

	err := validator.Validate(logger, order)
	assert.NoError(t, err)
}

func TestValidate_NilLogger(t *testing.T) {
	order := createValidOrder()

	// Должен работать без логгера
	err := validator.Validate(nil, order)
	assert.NoError(t, err)
}

func TestValidate_OrderFields(t *testing.T) {
	tests := []struct {
		name        string
		modifyOrder func(*models.Order)
		wantError   string
	}{
		{
			name: "empty order_uid",
			modifyOrder: func(o *models.Order) {
				o.OrderUID = ""
			},
			wantError: "order_uid is required",
		},
		{
			name: "whitespace order_uid",
			modifyOrder: func(o *models.Order) {
				o.OrderUID = "   \t\n  "
			},
			wantError: "order_uid is required",
		},
		{
			name: "empty track_number",
			modifyOrder: func(o *models.Order) {
				o.TrackNumber = ""
			},
			wantError: "track_number is required",
		},
		{
			name: "empty entry",
			modifyOrder: func(o *models.Order) {
				o.Entry = ""
			},
			wantError: "entry is required",
		},
		{
			name: "empty delivery_service",
			modifyOrder: func(o *models.Order) {
				o.DeliveryService = ""
			},
			wantError: "delivery_service is required",
		},
		{
			name: "zero date_created",
			modifyOrder: func(o *models.Order) {
				o.DateCreated = time.Time{}
			},
			wantError: "date_created is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := createValidOrder()
			tt.modifyOrder(order)

			err := validator.Validate(nil, order)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, validator.ErrBadMessage))
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_DeliveryFields(t *testing.T) {
	tests := []struct {
		name        string
		modifyOrder func(*models.Order)
		wantError   string
	}{
		{
			name: "empty delivery name",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Name = ""
			},
			wantError: "delivery.name is required",
		},
		{
			name: "empty delivery phone",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Phone = ""
			},
			wantError: "delivery.phone is required",
		},
		{
			name: "empty delivery zip",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Zip = ""
			},
			wantError: "delivery.zip is required",
		},
		{
			name: "empty delivery city",
			modifyOrder: func(o *models.Order) {
				o.Delivery.City = ""
			},
			wantError: "delivery.city is required",
		},
		{
			name: "empty delivery address",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Address = ""
			},
			wantError: "delivery.address is required",
		},
		{
			name: "empty delivery region",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Region = ""
			},
			wantError: "delivery.region is required",
		},
		{
			name: "empty delivery email",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Email = ""
			},
			wantError: "delivery.email is required",
		},
		{
			name: "invalid email format",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Email = "not-an-email"
			},
			wantError: "delivery.email has invalid format",
		},
		{
			name: "invalid email - missing @",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Email = "testgmail.com"
			},
			wantError: "delivery.email has invalid format",
		},
		{
			name: "invalid email - missing domain",
			modifyOrder: func(o *models.Order) {
				o.Delivery.Email = "test@"
			},
			wantError: "delivery.email has invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := createValidOrder()
			tt.modifyOrder(order)

			err := validator.Validate(nil, order)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, validator.ErrBadMessage))
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_PaymentFields(t *testing.T) {
	tests := []struct {
		name        string
		modifyOrder func(*models.Order)
		wantError   string
	}{
		{
			name: "empty payment transaction",
			modifyOrder: func(o *models.Order) {
				o.Payment.Transaction = ""
			},
			wantError: "payment.transaction is required",
		},
		{
			name: "payment transaction not equal order_uid",
			modifyOrder: func(o *models.Order) {
				o.Payment.Transaction = "different_transaction"
			},
			wantError: "payment.transaction must equal order_uid",
		},
		{
			name: "empty payment currency",
			modifyOrder: func(o *models.Order) {
				o.Payment.Currency = ""
			},
			wantError: "payment.currency is required",
		},
		{
			name: "empty payment provider",
			modifyOrder: func(o *models.Order) {
				o.Payment.Provider = ""
			},
			wantError: "payment.provider is required",
		},
		{
			name: "zero payment_dt",
			modifyOrder: func(o *models.Order) {
				o.Payment.PaymentDT = 0
			},
			wantError: "payment.payment_dt must be positive",
		},
		{
			name: "negative payment_dt",
			modifyOrder: func(o *models.Order) {
				o.Payment.PaymentDT = -1
			},
			wantError: "payment.payment_dt must be positive",
		},
		{
			name: "negative amount",
			modifyOrder: func(o *models.Order) {
				o.Payment.Amount = -100
			},
			wantError: "payment.amount must be >= 0",
		},
		{
			name: "negative delivery_cost",
			modifyOrder: func(o *models.Order) {
				o.Payment.DeliveryCost = -50
			},
			wantError: "payment.delivery_cost must be >= 0",
		},
		{
			name: "negative goods_total",
			modifyOrder: func(o *models.Order) {
				o.Payment.GoodsTotal = -200
			},
			wantError: "payment.goods_total must be >= 0",
		},
		{
			name: "negative custom_fee",
			modifyOrder: func(o *models.Order) {
				o.Payment.CustomFee = -10
			},
			wantError: "payment.custom_fee must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := createValidOrder()
			tt.modifyOrder(order)

			err := validator.Validate(nil, order)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, validator.ErrBadMessage))
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_ItemsFields(t *testing.T) {
	tests := []struct {
		name        string
		modifyOrder func(*models.Order)
		wantError   string
	}{
		{
			name: "zero chrt_id",
			modifyOrder: func(o *models.Order) {
				o.Items[0].ChrtID = 0
			},
			wantError: "items[0]: chrt_id must be > 0",
		},
		{
			name: "negative chrt_id",
			modifyOrder: func(o *models.Order) {
				o.Items[0].ChrtID = -1
			},
			wantError: "items[0]: chrt_id must be > 0",
		},
		{
			name: "empty item track_number",
			modifyOrder: func(o *models.Order) {
				o.Items[0].TrackNumber = ""
			},
			wantError: "items[0]: track_number is required",
		},
		{
			name: "item track_number not equal order track_number",
			modifyOrder: func(o *models.Order) {
				o.Items[0].TrackNumber = "DIFFERENT_TRACK"
			},
			wantError: "items[0]: track_number must equal order.track_number",
		},
		{
			name: "negative price",
			modifyOrder: func(o *models.Order) {
				o.Items[0].Price = -100
			},
			wantError: "items[0]: price must be >= 0",
		},
		{
			name: "empty rid",
			modifyOrder: func(o *models.Order) {
				o.Items[0].Rid = ""
			},
			wantError: "items[0]: rid is required",
		},
		{
			name: "empty item name",
			modifyOrder: func(o *models.Order) {
				o.Items[0].Name = ""
			},
			wantError: "items[0]: name is required",
		},
		{
			name: "negative sale",
			modifyOrder: func(o *models.Order) {
				o.Items[0].Sale = -10
			},
			wantError: "items[0]: sale must be >= 0",
		},
		{
			name: "negative total_price",
			modifyOrder: func(o *models.Order) {
				o.Items[0].TotalPrice = -50
			},
			wantError: "items[0]: total_price must be >= 0",
		},
		{
			name: "zero nm_id",
			modifyOrder: func(o *models.Order) {
				o.Items[0].NmID = 0
			},
			wantError: "items[0]: nm_id must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := createValidOrder()
			tt.modifyOrder(order)

			err := validator.Validate(nil, order)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, validator.ErrBadMessage))
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_MultipleItems(t *testing.T) {
	order := createValidOrder()
	// Добавляем второй item с ошибками
	order.Items = append(order.Items, models.Item{
		ChrtID:      0,             // Ошибка
		TrackNumber: "WRONG_TRACK", // Ошибка
		Price:       100,
		Rid:         "rid2",
		Name:        "Item 2",
		Sale:        0,
		TotalPrice:  100,
		NmID:        123456,
		Brand:       "Brand",
		Status:      202,
	})

	err := validator.Validate(nil, order)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "items[1]: chrt_id must be > 0")
	assert.Contains(t, err.Error(), "items[1]: track_number must equal order.track_number")
}
