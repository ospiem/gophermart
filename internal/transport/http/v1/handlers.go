package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/models/status"
	"github.com/ospiem/gophermart/internal/transport/http/v1/middleware/auth"
	"golang.org/x/crypto/bcrypt"
)

const handler = "handler"
const applicationJSON = "application/json"
const invalidContentType = "invalid content-type"
const cannotGetUser = "cannot get user from db"
const authorization = "Authorization"
const contentType = "Content-Type"
const luhnAlgoDivisor = 10
const invalidBody = "Invalid body"
const tokenExp = time.Hour * 336
const invalitContentTypeNotJSON = "Invalid Content-Type, expected application/json"

var ErrOrderBelongsAnotherUser = errors.New("the order belongs to another user")
var ErrOrderExists = errors.New("order exists")
var ErrInsufficientPoints = errors.New("insufficient points ")

func (a *API) registerUser(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "registerUser").Logger()

	if r.Header.Get(contentType) != applicationJSON {
		http.Error(w, invalitContentTypeNotJSON, http.StatusBadRequest)
		logger.Debug().Msg(invalidContentType)
		return
	}

	ctx := r.Context()
	credentials := models.Credentials{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&credentials); err != nil {
		http.Error(w, invalidBody, http.StatusBadRequest)
		return
	}
	if credentials.Login == "" || credentials.Pass == "" {
		http.Error(w, invalidBody, http.StatusBadRequest)
		return
	}

	hash, err := hashPass(credentials.Pass)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg(cannotGetUser)
		return
	}

	if err := a.storage.InsertUser(ctx, credentials.Login, hash, a.log); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "That username is taken. Try another", http.StatusConflict)
			return
		}
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot insert new user")
		return
	}

	token, err := buildJWTString(credentials.Login, a.cfg.JWTSecretKey)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot build token string")
		return
	}

	w.Header().Set(authorization, token)
	w.WriteHeader(http.StatusOK)
}

func (a *API) authUser(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "authUser").Logger()

	if r.Header.Get(contentType) != applicationJSON {
		http.Error(w, invalitContentTypeNotJSON, http.StatusBadRequest)
		logger.Debug().Msg(invalidContentType)
		return
	}

	ctx := r.Context()
	loginCreds := models.Credentials{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&loginCreds); err != nil {
		http.Error(w, invalidBody, http.StatusBadRequest)
		return
	}
	dbCreds, err := a.storage.SelectCreds(ctx, loginCreds.Login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg(cannotGetUser)
		return
	}

	if err := compareHash(dbCreds.Pass, loginCreds.Pass); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := buildJWTString(loginCreds.Login, a.cfg.JWTSecretKey)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot build token string")
		return
	}

	w.Header().Set(authorization, token)
	w.WriteHeader(http.StatusOK)
}

func (a *API) insertOrder(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "insertOrder").Logger()

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
	orderID := string(body)
	if err = validByLuhnAlgo(orderID); err != nil {
		http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		logger.Debug().Err(err).Msg("")
		return
	}

	login, ok := r.Context().Value(auth.ContextLoginKey).(string)
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	order := models.Order{
		ID:       orderID,
		Status:   status.NEW,
		Username: login,
	}

	err = checkOrderExists(ctx, a.storage, order.ID, order.Username)
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

	login, ok := r.Context().Value(auth.ContextLoginKey).(string)
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	orders, err := a.storage.SelectOrders(ctx, login)
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
	w.WriteHeader(http.StatusOK)
}

func (a *API) orderWithdraw(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "orderWithdraw").Logger()
	if r.Header.Get(contentType) != applicationJSON {
		http.Error(w, invalitContentTypeNotJSON, http.StatusBadRequest)
		logger.Debug().Msg(invalidContentType)
		return
	}
	ctx := r.Context()
	w.Header().Set(contentType, applicationJSON)

	login, ok := r.Context().Value(auth.ContextLoginKey).(string)
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	withdraw := models.Withdraw{}
	withdraw.User = login
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&withdraw); err != nil {
		http.Error(w, invalidBody, http.StatusBadRequest)
		logger.Error().Err(err)
		return
	}

	if err := validByLuhnAlgo(withdraw.OrderNumber); err != nil {
		http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		logger.Debug().Err(err).Msg("")
		return
	}

	if err := proceedWithdraw(ctx, a, withdraw); err != nil {
		if errors.Is(err, ErrInsufficientPoints) {
			http.Error(w, "Insufficient points", http.StatusPaymentRequired)
			return
		}
		if errors.Is(pgx.ErrNoRows, err) {
			http.Error(w, "Insufficient points", http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot proceed withdraw")
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) getWithdrawals(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "getWithdrawals").Logger()
	ctx := r.Context()
	w.Header().Set(contentType, applicationJSON)

	login, ok := r.Context().Value(auth.ContextLoginKey).(string)
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	withdraws, err := a.storage.SelectWithdraws(ctx, login)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot get withdraws")
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(withdraws); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot marshal withdraws")
	}
}

func (a *API) getBalance(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str(handler, "getBalance").Logger()
	ctx := r.Context()
	w.Header().Set(contentType, applicationJSON)
	login, ok := r.Context().Value(auth.ContextLoginKey).(string)
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	user, err := a.storage.SelectUserBalance(ctx, login)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot get balance")
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(user); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot encode balance")
	}
}

func buildJWTString(login string, key string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		models.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
			},
			Login: login,
		})
	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		return "", fmt.Errorf("cannot build token string: %w", err)
	}
	return tokenString, nil
}

func validByLuhnAlgo(orderNum string) error {
	re := regexp.MustCompile(`^\d+$`)
	if !re.MatchString(orderNum) {
		return fmt.Errorf("order number contains non-digit characters: %s", orderNum)
	}

	var sum int
	isSecond := false

	for _, char := range orderNum {
		d, err := strconv.Atoi(string(char))
		if err != nil {
			return fmt.Errorf("cannot convert string to int: %w", err)
		}

		if isSecond {
			d *= 2
		}
		sum += d / luhnAlgoDivisor
		sum += d % luhnAlgoDivisor
		isSecond = !isSecond
	}

	if !(sum%luhnAlgoDivisor == 0) {
		return fmt.Errorf("the sum of the digits is not divisible by 10")
	}
	return nil
}

func checkOrderExists(ctx context.Context, s storage, newOrder string, newUser string) error {
	selectOrder, err := s.SelectOrder(ctx, newOrder)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("cannot select the order: %w", err)
		}
		return nil
	}
	if selectOrder.Username == newUser {
		return ErrOrderExists
	}

	return ErrOrderBelongsAnotherUser
}

func hashPass(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("cannot generate hash: %w", err)
	}
	return string(hash), nil
}

func compareHash(dbHash string, reqPass string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(reqPass)); err != nil {
		return fmt.Errorf("cannot compare passwords: %w", err)
	}
	return nil
}

func proceedWithdraw(ctx context.Context, a *API, withdraw models.Withdraw) error {
	_, err := a.storage.SelectOrder(ctx, withdraw.OrderNumber)
	if err != nil {
		return fmt.Errorf("cannot select order: %w", err)
	}

	u, err := a.storage.SelectUserBalance(ctx, withdraw.User)
	if err != nil {
		return fmt.Errorf("cannot select user: %w", err)
	}
	if u.Balance < withdraw.Sum {
		return ErrInsufficientPoints
	}

	if err := a.storage.InsertWithdraw(ctx, withdraw, a.log); err != nil {
		return fmt.Errorf("cannot insert withdraw: %w", err)
	}
	return nil
}
