package usecase

import (
	"errors"
	"fmt"
	"github.com/OlegMzhelskiy/gophermart/pkg/accrual"
	"github.com/OlegMzhelskiy/gophermart/pkg/validate"
	"log"
	"time"

	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
)

var (
	ErrOrderAlreadyUploadThisUser    = errors.New("order already upload this user")
	ErrOrderAlreadyUploadAnotherUser = errors.New("order already upload another user")
	ErrInvalidOrderNumber            = errors.New("invalid order number")
	ErrNotEnoughFunds                = errors.New("not enough funds in the account")
	ErrWithdrawAlreadyExist          = errors.New("withdraw on this order already exist")
)

type OrderUseCase struct {
	repo             storage.Repository
	processingOrders []models.OrderNumber
	chProcOrder      chan models.OrderNumber
	accrual          accrual.Accrualer
}

func NewOrderUseCase(repo storage.Repository, done chan struct{}, asAdr string) OrderUseCase {
	u := OrderUseCase{
		repo:        repo,
		chProcOrder: make(chan models.OrderNumber, 100),
		accrual:     accrual.NewSystem(asAdr),
	}

	var err error
	u.processingOrders, err = u.repo.GetOrdersWithStatus(models.OrderStatusProcessing, models.OrderStatusNew)
	if err != nil {
		log.Printf("get order with status failed: %s", err)
	}

	go u.workerGettingOrderStatus(done)

	return u
}

func (u OrderUseCase) UploadOrder(order models.Order) error {
	if order.Number == "" || !validate.CheckLuna(order.Number) {
		return ErrInvalidOrderNumber
	}
	orderDB, err := u.repo.GetOrderByNumber(order.Number)
	if err != nil && err != storage.ErrOrderNotFound {
		return fmt.Errorf("get order by number failed: %w", err)
	}
	if err == storage.ErrOrderNotFound {
		err = u.repo.CreateOrder(order)
		if err != nil {
			return fmt.Errorf("create order failed: %w", err)
		}
		//send for processing
		go func(number models.OrderNumber) {
			isCalc, err := u.UpdateOrderInfoFromAccrual(number)
			if err != nil || !isCalc {
				u.chProcOrder <- number //add to queue
			}
		}(order.Number)

		return nil
	}
	if orderDB.UserID == order.UserID {
		return ErrOrderAlreadyUploadThisUser
	} else {
		return ErrOrderAlreadyUploadAnotherUser
	}
}

func (u OrderUseCase) GetOrderList(userID string) ([]models.Order, error) {
	return u.repo.GetOrderListByUserID(userID)
}

func (u OrderUseCase) GetWithdrawals(userID string) ([]models.OrderWithdraw, error) {
	userWith, err := u.repo.GetWithdrawalsListByUserID(userID)
	if err != nil {
		return userWith, fmt.Errorf("getting user's withdrawals failed: %w", err)
	}
	return userWith, nil
}

func (u OrderUseCase) Withdraw(userID string, withdraw models.WithdrawRequest) error {
	if withdraw.OrderNumber == "" { //|| !checkLuna(withdraw.OrderNumber) {
		return ErrInvalidOrderNumber
	}
	bal, err := u.repo.GetBalanceByUserID(userID)
	if err != nil {
		return fmt.Errorf("get balance failed: %w", err)
	}
	if bal < withdraw.Sum {
		return ErrNotEnoughFunds
	}
	err = u.repo.CreateWithdraw(userID, withdraw)
	if err != nil {
		if errors.Is(err, storage.ErrWithdrawAlreadyExist) {
			return ErrWithdrawAlreadyExist
		}
		return fmt.Errorf("create withdraw failed: %w", err)
	}
	return nil
}

// UpdateOrderInfoFromAccrual return (isCalculated, error)
func (u *OrderUseCase) UpdateOrderInfoFromAccrual(number models.OrderNumber) (bool, error) {
	req, err := u.accrual.GetOrderStatus(number)
	if err != nil {
		return false, err
	}
	var isCalc bool
	var order models.Order
	if req.Status != models.OrderAccrualStatusRegistered {
		if req.Status == models.OrderAccrualStatusProcessing {
			order, err = u.repo.GetOrderByNumber(number)
			if err != nil {
				return false, fmt.Errorf("get order by number failed: %w", err)
			}
			if order.Status != models.OrderStatusNew {
				return false, nil
			}
			order.Status = models.OrderStatus(req.Status)
		} else {
			isCalc = true
			order = models.Order{
				Number:  number,
				Accrual: req.Sum,
				Status:  models.OrderStatus(req.Status)}
		}
		return isCalc, u.repo.UpdateOrder(order)
	}
	return false, nil
}

func (u *OrderUseCase) workerGettingOrderStatus(done chan struct{}) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-done:
			fmt.Println("quit goroutine getting order status")
			return
		case <-ticker.C:
			for i, v := range u.processingOrders {
				isCalc, err := u.UpdateOrderInfoFromAccrual(v)
				if err == nil && isCalc {
					u.processingOrders = append(u.processingOrders[:i], u.processingOrders[i+1:]...) //remove from queue
				}
			}
		case number, ok := <-u.chProcOrder:
			if !ok {
				return
			}
			u.processingOrders = append(u.processingOrders, number) //add to queue
		}
	}
}
