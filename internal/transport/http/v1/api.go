package v1

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/ospiem/gophermart/internal/config"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/tools"
	"github.com/rs/zerolog"
)

type storage interface {
	InsertOrder(ctx context.Context, order models.Order, logger zerolog.Logger) error
	SelectOrder(ctx context.Context, num uint64) (models.Order, error)
	SelectOrders(ctx context.Context) ([]models.Order, error)
	SelectUser(ctx context.Context, login string) (models.User, error)
	InsertUser(ctx context.Context, login string, hash string, l zerolog.Logger) error
}

type API struct {
	storage storage
	log     zerolog.Logger
	cfg     config.Config
}

func New(cfg config.Config, s storage, l zerolog.Logger) *API {
	tools.SetGlobalLogLevel(cfg.LogLevel)
	return &API{
		cfg:     cfg,
		storage: s,
		log:     l,
	}
}

func (a *API) registerAPI() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", a.registerUser)
		r.Post("/login", a.authUser)
		r.Post("/orders", a.uploadOrder)
		r.Get("/orders", a.getOrders)
		r.Get("/withdrawals", a.getWithdrawals)

		r.Route("/balance", func(r chi.Router) {
			r.Get("/", a.getBalance)
			r.Post("/withdraw", a.orderWithdraw)
		})
	})

	return r
}

func (a *API) InitServer() *http.Server {
	a.log.Info().Msgf("Starting server on %s", a.cfg.Endpoint)

	r := a.registerAPI()
	return &http.Server{
		Addr:    a.cfg.Endpoint,
		Handler: r,
	}
}
