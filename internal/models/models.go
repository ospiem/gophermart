package models

import (
	"time"

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
	Login     string `json:"login"`
	Pass      string `json:"password"`
	Balance   float32
	Withdrawn float32
}
