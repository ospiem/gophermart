package models

import (
	"github.com/ospiem/gophermart/internal/models/status"
)

type Order struct {
	UploadedAt string
	Status     status.Status
	Number     uint64
	Accrual    uint64
}
