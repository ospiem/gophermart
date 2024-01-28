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
	ID        uint64
	Accrual   float32
}

type User struct {
	Credentials
	Balance   float32
	Withdrawn float32
}

type Credentials struct {
	Login string `json:"login"`
	Pass  string `json:"password"`
}

type Claims struct {
	jwt.RegisteredClaims
	Login string
}
