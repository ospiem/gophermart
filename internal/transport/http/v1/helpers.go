package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const luhnAlgoDivisor = 10

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

func isOrderExists(ctx context.Context, s storage, num uint64) (bool, error) {
	_, err := s.SelectOrder(ctx, num)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
		return false, nil
	}
	return true, nil
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
