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
	//UserExist(login string) (bool, error)
	//GetUserHashPassword(login string) (string, error)
	GetOrderByNumber(number string) (models.Order, error)
	CreateOrder(order models.Order) error
	GetOrderListByUserID(userID string) ([]models.Order, error)
	GetBalanceByUserID(userID string) (models.SumScore, error)
	GetWithdrawalsByUserID(userID string) (models.SumScore, error)
	//GetBalanceAndWithdrawalsByUserID(userID string) (models.UserBalance, error)
	CreateWithdraw(userID string, withdraw models.WithdrawRequest) error
	GetWithdrawalsListByUserID(userID string) ([]models.OrderWithdraw, error)
}