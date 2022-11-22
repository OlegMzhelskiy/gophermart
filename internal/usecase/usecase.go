package usecase

import (
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
)

type UseCases struct {
	User  UserUseCase
	Order OrderUseCase
}

func NewUseCases(repo storage.Repository, done chan struct{}, asAdr string) *UseCases {
	return &UseCases{
		User:  UserUseCase{repo: repo},
		Order: NewOrderUseCase(repo, done, asAdr),
	}
}

func (u UseCases) CloseRepo() {
	u.User.repo.Close()
	u.Order.repo.Close()
}
