package models

type User struct {
	ID                string `db:"id" swaggerignore:"true"`
	Login             string `db:"login" validate:"required" example:"user777"`
	Password          string `validate:"required" example:"qwerty12345"`
	EncryptedPassword string `db:"encrypted_password" swaggerignore:"true"`
}

type SumScore float64

type UserBalance struct {
	User      User     `json:"-"`
	Balance   SumScore `json:"current" db:"balance" example:"1950"`
	Withdrawn SumScore `json:"withdrawn" db:"withdraw" example:"710"`
}
