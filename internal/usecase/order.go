package usecase

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
	"github.com/OlegMzhelskiy/gophermart/pkg"
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
	ProcessingOrders []models.OrderNumber
	chProcOrder      chan models.OrderNumber
	accrualSysAdr    string
}

func NewOrderUseCase(repo storage.Repository, done chan struct{}, asAdr string) OrderUseCase {
	u := OrderUseCase{
		repo:          repo,
		chProcOrder:   make(chan models.OrderNumber, 100),
		accrualSysAdr: asAdr,
	}

	var err error
	u.ProcessingOrders, err = u.repo.GetOrdersWithStatus(models.OrderStatusProcessing, models.OrderStatusNew)
	if err != nil {
		log.Printf("get order with status failed: %s", err)
	}

	go u.WorkerGettingOrderStatus(done)

	return u
}

func (u OrderUseCase) UploadOrder(order models.Order) error {
	//if order.Number == "" {
	//	return ErrInvalidOrderNumber
	//}
	if order.Number == "" || !pkg.CheckLuna(order.Number) {
		return ErrInvalidOrderNumber
	}
	//_, err := u.repo.GetOrderByNumber(order.Number)
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
	//TODO:
	//	(нужно блокировать баланс на момент списания чтобы исключить одновременное списание 2 хендлерами)
	return nil
}

func GetOrderStatusFromAccrual(number models.OrderNumber, accrualSysAdr string) (models.AccrualRequest, error) {
	request := models.AccrualRequest{}
	//req, err := http.NewRequest("POST", , nil)
	res, err := http.Get(fmt.Sprintf("%s/api/orders/%s", accrualSysAdr, number))
	if err != nil {
		log.Printf("error to request accrual system: %s", err)
		return request, err
	}
	if res.StatusCode != 200 {
		return request, errors.New("error to request accrual system: " + res.Status)
	}
	err = json.NewDecoder(res.Body).Decode(&request)
	defer res.Body.Close()
	if err != nil {
		//resBody, err := io.ReadAll(body)
		log.Printf("error to request accrual system: %s", err)
		return request, err
	}
	return request, nil
}

// UpdateOrderInfoFromAccrual return (isCalculated, error)
func (u *OrderUseCase) UpdateOrderInfoFromAccrual(number models.OrderNumber) (bool, error) {
	req, err := GetOrderStatusFromAccrual(number, u.accrualSysAdr)
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

func (u *OrderUseCase) WorkerGettingOrderStatus(done chan struct{}) {
	//var number models.OrderNumber
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-done:
			fmt.Println("quit goroutine getting order status")
			return
		case <-ticker.C:
			for i, v := range u.ProcessingOrders {
				isCalc, err := u.UpdateOrderInfoFromAccrual(v)
				if err == nil && isCalc {
					u.ProcessingOrders = append(u.ProcessingOrders[:i], u.ProcessingOrders[i+1:]...) //remove from queue
				}
			}
		case number := <-u.chProcOrder:
			u.ProcessingOrders = append(u.ProcessingOrders, number) //add to queue
		}
	}
}
