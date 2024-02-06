package v1

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/ospiem/gophermart/internal/config"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/tools"
	"github.com/ospiem/gophermart/internal/transport/http/v1/middleware/auth"
	"github.com/ospiem/gophermart/internal/transport/http/v1/middleware/logger"
	"github.com/rs/zerolog"
)

type storage interface {
	InsertOrder(ctx context.Context, order models.Order, logger zerolog.Logger) error
	SelectOrder(ctx context.Context, num string) (models.Order, error)
	SelectOrders(ctx context.Context, user string) ([]models.Order, error)
	SelectCreds(ctx context.Context, login string) (models.Credentials, error)
	SelectUserBalance(ctx context.Context, login string) (models.UserBalance, error)
	InsertUser(ctx context.Context, login string, hash string, l zerolog.Logger) error
	InsertWithdraw(ctx context.Context, withdraw models.Withdraw, l zerolog.Logger) error
	SelectWithdraws(ctx context.Context, login string) ([]models.WithdrawResponse, error)
}

type API struct {
	storage storage
	log     zerolog.Logger
	cfg     config.Config
}

func New(cfg *config.Config, s storage, l *zerolog.Logger) *API {
	tools.SetGlobalLogLevel(cfg.LogLevel)
	return &API{
		cfg:     *cfg,
		storage: s,
		log:     *l,
	}
}

func (a *API) registerAPI() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(logger.RequestLogger(a.log))

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", a.registerUser)
		r.Post("/login", a.authUser)

		r.Group(func(r chi.Router) {
			r.Use(auth.JWTAuthorization(a.cfg.JWTSecretKey))
			r.Post("/orders", a.postOrder)
			r.Get("/orders", a.getOrders)
			r.Get("/withdrawals", a.getWithdrawals)

			r.Route("/balance", func(r chi.Router) {
				r.Get("/", a.getBalance)
				r.Post("/withdraw", a.orderWithdraw)
			})
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
