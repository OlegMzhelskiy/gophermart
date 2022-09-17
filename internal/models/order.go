package models

import (
	"encoding/json"
	"time"
)

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessed  = "PROCESSED"
)

type Order struct {
	Number     string    `json:"number" db:"number"`
	Status     string    `json:"status" db:"status"`
	Accrual    float64   `json:"accrual,omitempty" db:"sum"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	UserID     string    `json:"-" db:"user_id"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	type OrderAlias Order
	aliasValue := struct {
		OrderAlias
		UplAt string `json:"uploaded_at"`
	}{
		OrderAlias: OrderAlias(o),
		UplAt:      o.UploadedAt.Format(time.RFC3339),
	}
	return json.Marshal(aliasValue) // вызываем стандартный Marshal
}

type OrderWithdraw struct {
	OrderNumber string    `json:"order" db:"order_number"`
	Sum         float64   `json:"sum" db:"sum"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}

type WithdrawRequest struct {
	OrderNumber string   `json:"order"`
	Sum         SumScore `json:"sum"`
}
