package v1

import (
	"io"
	"net/http"
	"strconv"

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

	orderID, err := strconv.ParseUint(string(body), 10, 64)
	if err != nil {
		http.Error(w, "", http.StatusUnprocessableEntity)
		logger.Error().Err(err).Msg("cannot convert text to int")
		return
	}

	order := models.Order{
		ID:     orderID,
		Status: status.NEW,
	}

	exists, err := isOrderExists(ctx, a.storage, orderID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error().Err(err).Msg("cannot check if order exists")
		return
	}
	if exists {
		w.WriteHeader(http.StatusOK)
		logger.Debug().Msg("order already exists")
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

}

func (a *API) getWithdrawals(w http.ResponseWriter, r *http.Request) {

}

func (a *API) getBalance(w http.ResponseWriter, r *http.Request) {

}

func (a *API) orderWithdraw(w http.ResponseWriter, r *http.Request) {

}
