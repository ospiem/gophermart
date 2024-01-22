package models

import (
	"time"

	"github.com/ospiem/gophermart/internal/models/status"
)

type Order struct {
	CreatedAt time.Time
	Status    status.Status
	ID        uint64
	Accrual   uint64
}
