package models

import (
	"github.com/ospiem/gophermart/internal/models/status"
)

type Order struct {
	Number     uint64
	Status     status.Status
	Accrual    uint64
	UploadedAt string
}
