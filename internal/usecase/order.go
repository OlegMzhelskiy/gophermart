package usecase

import (
	"errors"
	"fmt"
	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
	"strconv"
)

var (
	ErrOrderAlreadyUploadThisUser    = errors.New("order already upload this user")
	ErrOrderAlreadyUploadAnotherUser = errors.New("order already upload another user")
	ErrInvalidOrderNumber            = errors.New("invalid order number")
	ErrNotEnoughFunds                = errors.New("not enough funds in the account")
	ErrWithdrawAlreadyExist          = errors.New("withdraw on this order already exist")
)

type OrderUseCase struct {
	repo storage.Repository
}

func (u OrderUseCase) UploadOrder(order models.Order) error {
	if order.Number == "" {
		return ErrInvalidOrderNumber
	}
	// TODO: проверка номера алгоритмом Луна
	//if order.Number == "" || !checkLuna(order.Number) {
	//	return ErrInvalidOrderNumber
	//}
	orderDB, err := u.repo.GetOrderByNumber(order.Number)
	if err != nil && err != storage.ErrOrderNotFound {
		return fmt.Errorf("get order by number failed: %w", err)
	}
	if err == storage.ErrOrderNotFound {
		err := u.repo.CreateOrder(order)
		if err != nil {
			return fmt.Errorf("create order failed: %w", err)
		}
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
	// проверка номера
	// проверка суммы достаточной для списания
	//	(нужно блокировать баланс на момент списания чтобы исключить одновременное списание 2 хендлерами)
	// ...
	// создание записи в БД
	return nil
}

func checkLuna(num string) bool {
	var sum int
	var n int
	var err error
	lenNum := len(num)
	even := lenNum % 2
	fmt.Printf("len num: %d\n", lenNum)
	for i, s := range num {
		n, err = strconv.Atoi(string(s))
		if err != nil {
			return false
		}
		//fmt.Println(string(s))
		if i%2 == even {
			n = n * 2
			if n > 9 {
				n = n - 9
			}
		}
		sum = sum + n
		//fmt.Printf("%d+", n)
	}
	check := sum % 10
	//fmt.Printf("\nsum: %d\n", sum)
	//fmt.Printf("check: %d\n", check)
	if lenNum%2 == 0 {
		if check == 0 {
			return true
		}
	} else {
		return check == n
	}
	return false
}
