package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/ospiem/gophermart/internal/models"
	"golang.org/x/crypto/bcrypt"
)

const luhnAlgoDivisor = 10

var ErrOrderExists = errors.New("order exists")
var ErrOrderBelongsAnotherUser = errors.New("the order belongs to another user")

func isValidByLuhnAlgo(numbers []int) bool {
	var sum int
	isSecond := false
	for _, d := range numbers {
		if isSecond {
			d *= 2
		}
		sum += d / luhnAlgoDivisor
		sum += d % luhnAlgoDivisor
		isSecond = !isSecond
	}
	return sum%luhnAlgoDivisor == 0
}

func isOrderExists(ctx context.Context, s storage, newOrder models.Order) error {
	selectOrder, err := s.SelectOrder(ctx, newOrder.ID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("cannot select the order: %w", err)
		}
		return nil
	}
	if selectOrder.Username == newOrder.Username {
		return ErrOrderExists
	}

	return ErrOrderBelongsAnotherUser
}

func isUserExists(ctx context.Context, s storage, login string) (bool, error) {
	_, err := s.SelectUser(ctx, login)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("cannot select the user: %w", err)
		}
		return false, nil
	}
	return true, nil
}

func hashPass(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("cannot generate hash: %w", err)
	}
	return string(hash), nil
}

func compareHash(ctx context.Context, s storage, login string, pass string) error {
	user, err := s.SelectUser(ctx, login)
	if err != nil {
		return fmt.Errorf("cannot get user from db: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Pass), []byte(pass)); err != nil {
		return fmt.Errorf("cannot compare passwords: %w", err)
	}
	return nil
}

func marshalBalanceAndWithdrawn(user models.User, w http.ResponseWriter) error {
	enc := json.NewEncoder(w)

	err := enc.Encode(struct {
		Balance   float32 `json:"balance"`
		Withdrawn float32 `json:"withdrawn"`
	}{
		Balance:   user.Balance,
		Withdrawn: user.Withdrawn})
	if err != nil {
		return fmt.Errorf("cannot marshal balance and withdrawn: %w", err)
	}

	return nil
}
