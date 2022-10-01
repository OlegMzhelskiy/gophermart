package storage

import (
	"errors"
	"github.com/OlegMzhelskiy/gophermart/internal/models"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrOrderNotFound        = errors.New("order not found")
	ErrWithdrawAlreadyExist = errors.New("withdraw on this order already exist")
)

type Repository interface {
	Open() error
	Close()
	CreateUser(login, password string) (string, error)
	GetUserByLogin(login string) (models.User, error)
	GetOrderByNumber(number models.OrderNumber) (models.Order, error)
	CreateOrder(order models.Order) error
	GetOrderListByUserID(userID string) ([]models.Order, error)
	GetBalanceByUserID(userID string) (models.SumScore, error)
	GetWithdrawalsByUserID(userID string) (models.SumScore, error)
	CreateWithdraw(userID string, withdraw models.WithdrawRequest) error
	GetWithdrawalsListByUserID(userID string) ([]models.OrderWithdraw, error)
	GetOrdersWithStatus(status ...models.OrderStatus) ([]models.OrderNumber, error)
	UpdateOrder(models.Order) error
}
