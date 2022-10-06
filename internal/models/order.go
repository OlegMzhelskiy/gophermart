package models

import (
	"encoding/json"
	"time"
)

type OrderStatus string

const (
	OrderStatusNew               OrderStatus = "NEW"
	OrderStatusProcessing        OrderStatus = "PROCESSING"
	OrderStatusInvalid           OrderStatus = "INVALID"
	OrderStatusProcessed         OrderStatus = "PROCESSED"
	OrderAccrualStatusRegistered             = "REGISTERED"
	OrderAccrualStatusProcessing             = "PROCESSING"
	OrderAccrualStatusInvalid                = "INVALID"
	OrderAccrualStatusProcessed              = "PROCESSED"
)

type OrderNumber string

type Order struct {
	Number     OrderNumber `json:"number" db:"number"`
	Status     OrderStatus `json:"status" db:"status"`
	Accrual    SumScore    `json:"accrual,omitempty" db:"sum"`
	UserID     string      `json:"-" db:"user_id"`
	UploadedAt time.Time   `json:"uploaded_at" db:"uploaded_at"`
	UpdatedAt  time.Time   `json:"-" db:"updated_at"`
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
