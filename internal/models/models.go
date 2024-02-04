package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ospiem/gophermart/internal/models/status"
)

type Order struct {
	CreatedAt time.Time
	Status    status.Status
	Username  string
	ID        string
	Accrual   float32
}

type UserBalance struct {
	Balance   float32 `json:"balance"`
	Withdrawn float32 `json:"withdrawn"`
}

type User struct {
	Credentials
	UserBalance
}

type Credentials struct {
	Login string `json:"login"`
	Pass  string `json:"password"`
}

type Claims struct {
	jwt.RegisteredClaims
	Login string
}

type Withdraw struct {
	OrderNumber string  `json:"order"`
	User        string  `json:"-"`
	Sum         float32 `json:"sum"`
}

type WithdrawResponse struct {
	ProcessedAt time.Time `json:"processed_at"`
	Order       string    `json:"order"`
	Sum         float32   `json:"sum"`
}
