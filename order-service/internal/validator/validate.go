package validator

import (
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"order-service/internal/models"
	"strings"
)

var ErrBadMessage = errors.New("bad_message")

func Validate(log *slog.Logger, o *models.Order) error {
	var errs []string
	validateOrder(&errs, o)
	validateDelivery(&errs, &o.Delivery)
	validatePayment(&errs, &o.Payment, o)
	validateItems(&errs, o)

	if len(errs) > 0 {
		if log != nil {
			log.Warn("order validation failed",
				slog.String("order_uid", o.OrderUID),
				slog.Any("errors", errs),
			)
		}
		return fmt.Errorf("%w: %s", ErrBadMessage, strings.Join(errs, "; "))
	}
	return nil
}

func validateOrder(errs *[]string, o *models.Order) {
	if strings.TrimSpace(o.OrderUID) == "" {
		*errs = append(*errs, "order_uid is required")
	}
	if strings.TrimSpace(o.TrackNumber) == "" {
		*errs = append(*errs, "track_number is required")
	}
	if strings.TrimSpace(o.Entry) == "" {
		*errs = append(*errs, "entry is required")
	}
	if strings.TrimSpace(o.DeliveryService) == "" {
		*errs = append(*errs, "delivery_service is required")
	}
	if o.DateCreated.IsZero() {
		*errs = append(*errs, "date_created is required")
	}
}

func validateDelivery(errs *[]string, d *models.Delivery) {
	if strings.TrimSpace(d.Name) == "" {
		*errs = append(*errs, "delivery.name is required")
	}
	if strings.TrimSpace(d.Phone) == "" {
		*errs = append(*errs, "delivery.phone is required")
	}
	if strings.TrimSpace(d.Zip) == "" {
		*errs = append(*errs, "delivery.zip is required")
	}
	if strings.TrimSpace(d.City) == "" {
		*errs = append(*errs, "delivery.city is required")
	}
	if strings.TrimSpace(d.Address) == "" {
		*errs = append(*errs, "delivery.address is required")
	}
	if strings.TrimSpace(d.Region) == "" {
		*errs = append(*errs, "delivery.region is required")
	}
	if strings.TrimSpace(d.Email) == "" {
		*errs = append(*errs, "delivery.email is required")
	} else if !isValidEmail(d.Email) {
		*errs = append(*errs, "delivery.email has invalid format")
	}
}

func validatePayment(errs *[]string, p *models.Payment, o *models.Order) {
	if strings.TrimSpace(p.Transaction) == "" {
		*errs = append(*errs, "payment.transaction is required")
	}
	if p.Transaction != o.OrderUID {
		*errs = append(*errs, "payment.transaction must equal order_uid")
	}
	if strings.TrimSpace(p.Currency) == "" {
		*errs = append(*errs, "payment.currency is required")
	}
	if strings.TrimSpace(p.Provider) == "" {
		*errs = append(*errs, "payment.provider is required")
	}
	if p.PaymentDT <= 0 {
		*errs = append(*errs, "payment.payment_dt must be positive")
	}

	if p.Amount < 0 {
		*errs = append(*errs, "payment.amount must be >= 0")
	}
	if p.DeliveryCost < 0 {
		*errs = append(*errs, "payment.delivery_cost must be >= 0")
	}
	if p.GoodsTotal < 0 {
		*errs = append(*errs, "payment.goods_total must be >= 0")
	}
	if p.CustomFee < 0 {
		*errs = append(*errs, "payment.custom_fee must be >= 0")
	}
}

func validateItems(errs *[]string, o *models.Order) {
	for i, it := range o.Items {
		pfx := fmt.Sprintf("items[%d]", i)

		if it.ChrtID <= 0 {
			*errs = append(*errs, pfx+": chrt_id must be > 0")
		}
		if strings.TrimSpace(it.TrackNumber) == "" {
			*errs = append(*errs, pfx+": track_number is required")
		}
		if it.Price < 0 {
			*errs = append(*errs, pfx+": price must be >= 0")
		}
		if strings.TrimSpace(it.Rid) == "" {
			*errs = append(*errs, pfx+": rid is required")
		}
		if strings.TrimSpace(it.Name) == "" {
			*errs = append(*errs, pfx+": name is required")
		}
		if it.Sale < 0 {
			*errs = append(*errs, pfx+": sale must be >= 0")
		}
		if it.TotalPrice < 0 {
			*errs = append(*errs, pfx+": total_price must be >= 0")
		}
		if it.NmID <= 0 {
			*errs = append(*errs, pfx+": nm_id must be > 0")
		}

		if it.TrackNumber != o.TrackNumber {
			*errs = append(*errs, pfx+": track_number must equal order.track_number")
		}
	}
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
