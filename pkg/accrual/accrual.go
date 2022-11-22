package accrual

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/OlegMzhelskiy/gophermart/internal/models"
)

type Accrualer interface {
	GetOrderStatus(number models.OrderNumber) (AccrualRequest, error)
}

type AccrualSystem struct {
	addr string
}

type AccrualRequest struct {
	Order  string          `json:"order"`
	Status string          `json:"status"`
	Sum    models.SumScore `json:"accrual"`
}

func NewSystem(addr string) Accrualer {
	return &AccrualSystem{addr: addr}
}

func (a AccrualSystem) GetOrderStatus(number models.OrderNumber) (AccrualRequest, error) {
	request := AccrualRequest{}
	res, err := http.Get(fmt.Sprintf("%s/api/orders/%s", a.addr, number))
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
		log.Printf("error to request accrual system: %s", err)
		return request, err
	}
	return request, nil
}
