package models

type User struct {
	ID                string `db:"id"`
	Login             string `db:"login"`
	Password          string
	EncryptedPassword string `db:"encrypted_password"`
}

type SumScore float64

type UserBalance struct {
	User      User     `json:"-"`
	Balance   SumScore `json:"current" db:"balance"`
	Withdrawn SumScore `json:"withdrawn" db:"withdraw"`
}
