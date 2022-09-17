package usecase

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"

	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
)

const (
	tokenTTL  = 12 * time.Hour
	secretKey = "dSKklbjNiulfhvsdKmvdjvnuLQhnriludhnbmijlJHADSFodfgok"
)

var (
	ErrLoginIsEmpty           = errors.New("login is empty")
	ErrLoginAlreadyExists     = errors.New("login already exists")
	ErrInvalidLoginOrPassword = errors.New("invalid login or password")
	ErrPasswordTooShort       = errors.New("password must be longer than 5 characters")
)

type tokenClaims struct {
	jwt.StandardClaims
	UserID string `json:"user_id"`
}

type UserUseCase struct {
	repo storage.Repository
}

func (u UserUseCase) CreateUser(user *models.User) error {
	//validate login
	if len(user.Login) == 0 {
		return ErrLoginIsEmpty
	}
	//validate password
	if len(user.Password) < 6 {
		return ErrPasswordTooShort
	}
	//exist login
	//exist, err := u.repo.UserExist(user.Login)
	userBD, err := u.repo.GetUserByLogin(user.Login)
	if err != nil && err != storage.ErrUserNotFound {
		return fmt.Errorf("get user by id failed: %w", err)
	}
	if userBD.ID != "" {
		return ErrLoginAlreadyExists
	}
	//hashing password
	encryptedPas, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("hashing password failed: %w", err)
	}
	user.ID, err = u.repo.CreateUser(user.Login, string(encryptedPas)) //return userID
	if err != nil {
		return fmt.Errorf("create user failed: %w", err)
	}
	return nil
}

func (u UserUseCase) AuthUser(user *models.User) error {
	//hp, err := u.repo.GetUserHashPassword(user.Login)
	userBD, err := u.repo.GetUserByLogin(user.Login)
	if err != nil {
		return fmt.Errorf("getting user's password failed: %w", err)
	}
	if !comparePassword(userBD.EncryptedPassword, user.Password) {
		return ErrInvalidLoginOrPassword
	}
	user.ID = userBD.ID
	return nil
}

func (u UserUseCase) GetUserBalanceAndWithdrawals(userID string) (models.UserBalance, error) {
	//userBal, err := u.repo.GetBalanceAndWithdrawalsByUserID(userID)
	userBal := models.UserBalance{}
	bal, err := u.repo.GetBalanceByUserID(userID)
	if err != nil {
		return userBal, fmt.Errorf("getting user's balance failed: %w", err)
	}
	wd, err := u.repo.GetWithdrawalsByUserID(userID)
	if err != nil {
		return userBal, fmt.Errorf("getting user's withdrawals failed: %w", err)
	}
	userBal.Balance = bal
	userBal.Withdrawn = wd
	return userBal, nil
}

func comparePassword(hashPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password)) == nil
}

func (u UserUseCase) GenerateToken(user models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tokenTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserID: user.ID,
	})
	return token.SignedString([]byte(secretKey))
}

func (u UserUseCase) ParseToken(tokenString string) (bool, tokenClaims, error) {
	claims := tokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return false, claims, err
	}
	return token.Valid, claims, nil
}
