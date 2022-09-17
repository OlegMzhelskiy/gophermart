package usecase

import (
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
)

type UseCases struct {
	User  UserUseCase
	Order OrderUseCase
}

func NewUseCases(repo storage.Repository) *UseCases {
	return &UseCases{
		User:  UserUseCase{repo: repo},
		Order: OrderUseCase{repo: repo},
	}
}

func (u UseCases) CloseRepo() {
	u.User.repo.Close()
	//u.Order.repo.Close()
}
