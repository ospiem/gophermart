package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/ospiem/gophermart/internal/models"
	"golang.org/x/crypto/bcrypt"
)

const luhnAlgoDivisor = 10

var OrderExists = errors.New("order exists")
var OrderBelongsAnotherUser = errors.New("the order belongs to another user")

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
			return err
		}
		return nil
	}
	fmt.Printf("select: %s, new: %s", selectOrder.Username, newOrder.Username)
	if selectOrder.Username == newOrder.Username {
		return OrderExists
	}

	return OrderBelongsAnotherUser
}

func isUserExists(ctx context.Context, s storage, login string) (bool, error) {
	_, err := s.SelectUser(ctx, login)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, err
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
		return fmt.Errorf("cannot select user: %w", err)
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Pass), []byte(pass))
}
