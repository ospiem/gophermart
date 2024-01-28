package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/models/status"
	"golang.org/x/crypto/bcrypt"
)

const handler = "handler"
const applicationJSON = "application/json"
const invalidContentType = "invalid content-type"
const cannotGetUser = "cannot get user from db"
const authorization = "Authorization"
const contentType = "Content-Type"

func (a *API) registerUser(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "registerUser").Logger()

	if r.Header.Get(contentType) != applicationJSON {
		http.Error(w, "Invalid Content-Type, expected application/json", http.StatusBadRequest)
		logger.Debug().Msg(invalidContentType)
		return
	}

	ctx := r.Context()
	u := models.User{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&u); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	exists, err := isUserExists(ctx, a.storage, u.Login)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannotGetUser")
		return
	}
	if exists {
		w.WriteHeader(http.StatusConflict)
		return
	}

	hash, err := hashPass(u.Pass)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg(cannotGetUser)
		return
	}

	if err := a.storage.InsertUser(ctx, u.Login, hash, a.log); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg(cannotGetUser)
		return
	}

	//TODO: implement JWT
	w.Header().Set(authorization, u.Login)
	w.WriteHeader(http.StatusOK)
}

func (a *API) authUser(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "authUser").Logger()

	if r.Header.Get(contentType) != applicationJSON {
		http.Error(w, "Invalid Content-Type, expected application/json", http.StatusBadRequest)
		logger.Debug().Msg(invalidContentType)
		return
	}

	ctx := r.Context()
	u := models.User{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&u); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	exists, err := isUserExists(ctx, a.storage, u.Login)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg(cannotGetUser)
		return
	}
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := compareHash(ctx, a.storage, u.Login, u.Pass); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	//TODO: implement JWT
	w.Header().Set(authorization, u.Login)
	w.WriteHeader(http.StatusOK)
}

func (a *API) uploadOrder(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "uploadOrder").Logger()

	if r.Header.Get(contentType) != "text/plain" {
		http.Error(w, "Invalid Content-Type, expected text/plain", http.StatusBadRequest)
		logger.Debug().Msg(invalidContentType)
		return
	}

	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot read body")
	}

	orderID, err := strconv.ParseUint(string(body), 10, 64)
	if err != nil {
		http.Error(w, "", http.StatusUnprocessableEntity)
		logger.Error().Err(err).Msg("cannot convert text to int")
		return
	}

	//TODO: implement jwt
	user := r.Header.Get(authorization)
	order := models.Order{
		ID:       orderID,
		Status:   status.NEW,
		Username: user,
	}

	err = isOrderExists(ctx, a.storage, order)
	if err != nil {
		if errors.Is(err, ErrOrderExists) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, ErrOrderBelongsAnotherUser) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot check if order exists")
		return
	}

	if err = a.storage.InsertOrder(ctx, order, a.log); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot insert order to DB")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *API) getOrders(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "getOrders").Logger()
	ctx := r.Context()
	w.Header().Set(contentType, applicationJSON)

	user := r.Header.Get(authorization)
	orders, err := a.storage.SelectOrders(ctx, user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot get orders")
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		logger.Debug().Msg("user has no orders")
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(orders); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot get orders")
		return
	}
}

func (a *API) getWithdrawals(w http.ResponseWriter, r *http.Request) {

}

func (a *API) getBalance(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "getBalance").Logger()
	ctx := r.Context()
	w.Header().Set(contentType, applicationJSON)
	//TODO: implement JWT
	username := r.Header.Get(authorization)

	user, err := a.storage.SelectUser(ctx, username)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot get balance")
		return
	}
	if err = marshalBalanceAndWithdrawn(user, w); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot get balance")
	}

}

func (a *API) orderWithdraw(w http.ResponseWriter, r *http.Request) {

}

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
