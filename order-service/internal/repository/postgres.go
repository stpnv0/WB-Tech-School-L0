package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"order-service/internal/models"
)

var ErrNotFound = errors.New("order not found")

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// SaveOrder - в рамках одной транзакции вставляет в бд всю информацию о заказе
func (r *PostgresRepository) SaveOrder(ctx context.Context, order *models.Order) error {
	const op = "PostgresRepository.SaveOrder"

	queryOrder := `INSERT INTO orders
		(order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s: begin transaction %w", op, err)
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, queryOrder,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.Shardkey,
		order.SmID,
		order.DateCreated,
		order.OofShard,
	); err != nil {
		return fmt.Errorf("%s: insert order %w", op, err)
	}

	queryPayments := `INSERT INTO payments
		(order_id, transaction, request_id, currency, provider, amount,payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_id) DO UPDATE SET
			amount = EXCLUDED.amount,
			payment_dt = EXCLUDED.payment_dt`

	_, err = tx.Exec(ctx, queryPayments,
		order.OrderUID,
		order.Payment.Transaction,
		order.Payment.RequestID,
		order.Payment.Currency,
		order.Payment.Provider,
		order.Payment.Amount,
		order.Payment.PaymentDT,
		order.Payment.Bank,
		order.Payment.DeliveryCost,
		order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)
	if err != nil {
		return fmt.Errorf("%s: insert payments %w", op, err)
	}

	queryDelivery := `INSERT INTO delivery
		(order_id, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (order_id) DO UPDATE SET
			name = EXCLUDED.name,
			phone = EXCLUDED.phone,
			address = EXCLUDED.address`

	_, err = tx.Exec(ctx, queryDelivery,
		order.OrderUID,
		order.Delivery.Name,
		order.Delivery.Phone,
		order.Delivery.Zip,
		order.Delivery.City,
		order.Delivery.Address,
		order.Delivery.Region,
		order.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("%s: insert delivery %w", op, err)
	}

	queryItem := `INSERT INTO items
		(order_id, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`

	for _, i := range order.Items {
		_, err = tx.Exec(ctx, queryItem,
			order.OrderUID,
			i.ChrtID,
			i.TrackNumber,
			i.Price,
			i.Rid,
			i.Name,
			i.Sale,
			i.Size,
			i.TotalPrice,
			i.NmID,
			i.Brand,
			i.Status,
		)

		if err != nil {
			return fmt.Errorf("%s: insert items %w", op, err)
		}

	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	return nil
}

// GetOrderByUID - ищет в бд заказ по UID и возвращает всю структуру заказа
func (r *PostgresRepository) GetOrderByUID(ctx context.Context, orderUID string) (*models.Order, error) {
	const op = "PostgresRepository.GetOrderByUID"

	query := `SELECT 
    		o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id, 
    		o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, 
			p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		JOIN delivery d ON o.order_uid = d.order_uid
		JOIN payments p ON o.order_uid = p.order_uid
		WHERE o.order_uid = $1`

	var order models.Order
	order.OrderUID = orderUID
	err := r.db.QueryRow(ctx, query, orderUID).Scan(
		&order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.CustomerID,
		&order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
		&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
		&order.Payment.Amount, &order.Payment.PaymentDT, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	queryItems := `SELECT 
		chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items
		WHERE order_uid = $1`

	rows, err := r.db.Query(ctx, queryItems, orderUID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Item
		if err = rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		order.Items = append(order.Items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows err: %w", op, err)
	}

	return &order, nil
}

func (r *PostgresRepository) GetLastNOrders(ctx context.Context, numOrders int) ([]*models.Order, error) {
	const op = "PostgresRepository.GetLastNOrders"

	query := `SELECT 
		o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
        o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
        d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
        p.transaction, p.request_id, p.currency, p.provider, p.amount, 
        p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		LEFT JOIN delivery d ON o.order_uid = d.order_uid
		LEFT JOIN payments p ON o.order_uid = p.order_uid
		ORDER BY o.date_created DESC
		LIMIT $1`

	rows, err := r.db.Query(ctx, query, numOrders)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	//используем мапу, чтобы потом сопоставить items к заказам по orderUID
	orderMap := make(map[string]*models.Order)
	for rows.Next() {
		var order models.Order
		err = rows.Scan(
			&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
			&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
			&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
			&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
			&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
			&order.Payment.Amount, &order.Payment.PaymentDT, &order.Payment.Bank, &order.Payment.DeliveryCost,
			&order.Payment.GoodsTotal, &order.Payment.CustomFee,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}

		orderMap[order.OrderUID] = &order
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	if len(orderMap) == 0 {
		return []*models.Order{}, nil
	}

	orderUIDs := make([]string, 0, len(orderMap))
	for uid := range orderMap {
		orderUIDs = append(orderUIDs, uid)
	}

	queryItems := `
        SELECT 
            order_uid, chrt_id, track_number, price, rid, name, 
            sale, size, total_price, nm_id, brand, status
        FROM items
        WHERE order_id = ANY($1)`

	itemsRows, err := r.db.Query(ctx, queryItems, orderUIDs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer itemsRows.Close()

	for itemsRows.Next() {
		var orderUID string
		var item models.Item
		err = itemsRows.Scan(
			&orderUID, &item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan item failed: %w", op, err)
		}

		if order, exists := orderMap[orderUID]; exists {
			order.Items = append(order.Items, item)
		}
	}

	//конвертируем в слайс
	result := make([]*models.Order, 0, len(orderMap))
	for _, uid := range orderUIDs {
		if order, exists := orderMap[uid]; exists {
			result = append(result, order)
		}
	}

	return result, nil
}
