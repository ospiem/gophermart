package v1

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/models/status"
)

func (a *API) registerUser(w http.ResponseWriter, r *http.Request) {

}

func (a *API) loginUser(w http.ResponseWriter, r *http.Request) {

}

func (a *API) uploadOrder(w http.ResponseWriter, r *http.Request) {
	logger := a.log.With().Str("handler", "uploadOrder").Logger()

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Invalid Content-Type, expected text/plain", http.StatusBadRequest)
		logger.Debug().Msg("invalid content-type")
		return
	}

	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot read body")
	}

	on, err := strconv.ParseUint(string(body), 10, 64)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot convert string to int")
		return
	}

	order := models.Order{
		Number:     on,
		Status:     status.NEW,
		Accrual:    0,
		UploadedAt: time.Now().Format(time.RFC3339),
	}

	if err = a.storage.InsertOrder(ctx, order); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot insert order to DB")
		return
	}

}

func (a *API) getOrders(w http.ResponseWriter, r *http.Request) {

}

func (a *API) getWithdrawals(w http.ResponseWriter, r *http.Request) {

}

func (a *API) getBalance(w http.ResponseWriter, r *http.Request) {

}

func (a *API) orderWithdraw(w http.ResponseWriter, r *http.Request) {

}
