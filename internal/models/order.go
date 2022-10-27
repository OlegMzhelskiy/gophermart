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
	Number     OrderNumber `json:"number" db:"number" example:"9278923470"`
	Status     OrderStatus `json:"status" db:"status" example:"PROCESSED"`
	Accrual    SumScore    `json:"accrual,omitempty" db:"sum" example:"500"`
	UserID     string      `json:"-" db:"user_id"`
	UploadedAt time.Time   `json:"uploaded_at" db:"uploaded_at" example:"2021-12-10T15:15:45+03:00"`
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
	OrderNumber string   `json:"order" example:"9278923470"`
	Sum         SumScore `json:"sum" example:"125"`
}
