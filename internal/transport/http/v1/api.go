package v1

import (
	"log/slog"

	"github.com/go-chi/chi"
	"github.com/ospiem/gophermart/internal/config"
	"github.com/ospiem/gophermart/internal/storage"
)

type API struct {
	cfg     config.Config
	storage storage.Storage
	log     slog.Logger
}

func New(cfg config.Config, s storage.Storage, l slog.Logger) *API {
	return &API{
		cfg:     cfg,
		storage: s,
		log:     l,
	}
}

func (a *API) registerAPI() chi.Router {
	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", a.registerUser)
		r.Post("/login", a.loginUser)
		r.Post("/orders", a.uploadOrder)
		r.Get("/orders", a.getOrders)
		r.Get("/withdrawals", a.getWithdrawals)

		r.Route("/balance", func(r chi.Router) {
			r.Get("", a.getBalance)
			r.Post("/withdraw", a.orderWithdraw)
		})
	})

	return r
}
