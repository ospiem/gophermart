package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/ospiem/gophermart/internal/config"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/restclient"
	"github.com/ospiem/gophermart/internal/storage/postgres"
	api "github.com/ospiem/gophermart/internal/transport/http/v1"
	"github.com/rs/zerolog"
)

const timeoutShutdown = 25 * time.Second

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if err := run(logger); err != nil {
		logger.Fatal().Err(err)
	}
	logger.Info().Msg("Graceful shutdown completed successfully. All connections closed, and resources released.")
}

func run(logger zerolog.Logger) error {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelCtx()

	context.AfterFunc(ctx, func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), timeoutShutdown)
		defer cancelCtx()

		<-ctx.Done()
		logger.Fatal().Msg("failed to gracefully shutdown the service")
	})

	wg := &sync.WaitGroup{}
	defer func() {
		// When exiting the main function, we expect the completion of application components
		wg.Wait()
	}()

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("cannot initialize config: %w", err)
	}

	db, err := postgres.NewDB(ctx, cfg.DSN)
	if err != nil {
		return fmt.Errorf("cannot initialize PostgreSQL database: %w", err)
	}

	watchDB(ctx, wg, db, &logger)

	componentsErrs := make(chan error, 1)
	a := api.New(&cfg, db, &logger)
	srv := a.InitServer()
	manageServer(ctx, wg, srv, componentsErrs, &logger)

	r := restclient.New(&cfg, db, &logger)
	mu := &sync.RWMutex{}
	delayMap := make(map[string]int, 1)
	orderCH := make(chan models.Order, r.Cfg.Offset*r.Cfg.WorkersNum)
	manageClients(ctx, wg, r, mu, delayMap, orderCH)

	select {
	case <-ctx.Done():
	case err := <-componentsErrs:
		logger.Error().Err(err)
		cancelCtx()
	}

	return nil
}

func manageClients(ctx context.Context, wg *sync.WaitGroup, r *restclient.RestClient, mu *sync.RWMutex,
	delayMap map[string]int, orderCh chan models.Order) {
	wg.Add(1)

	for i := 0; i < r.Cfg.WorkersNum; i++ {
		wg.Add(1)
		go r.ProcessOrder(ctx, wg, mu, delayMap, orderCh)
	}

	go func() {
		defer wg.Done() // Decrement the WaitGroup when the goroutine exits
		for {
			select {
			case <-ctx.Done():
				r.Logger.Info().Msg("Stopped connection manager")
				return

			default:
				// Read delay from delayMap using RLock
				mu.RLock()
				delay := delayMap[restclient.DelayTime]
				mu.RUnlock()

				if delay != 0 {
					// If delay is not zero, sleep and reset delay in delayMap using Lock
					mu.Lock()
					time.Sleep(time.Duration(delay) * time.Second)
					delayMap["delayTime"] = 0
					mu.Unlock()
				}

				// Fetch orders from storage
				orders, err := r.Storage.SelectOrdersToProceed(ctx, r.Cfg.Offset)
				if err != nil {
					r.Logger.Err(err)
				}

				// Send orders to orderCh
				for _, o := range orders {
					orderCh <- o
				}
			}
		}
	}()
}

func watchDB(ctx context.Context, wg *sync.WaitGroup, db *postgres.DB, l *zerolog.Logger) {
	wg.Add(1)
	go func() {
		defer l.Info().Msg("DB has been closed")
		defer wg.Done()

		<-ctx.Done()

		db.Close()
	}()
}

func manageServer(ctx context.Context, wg *sync.WaitGroup, srv *http.Server, errs chan error, l *zerolog.Logger) {
	go func(errs chan<- error) {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			errs <- fmt.Errorf("listen and serve has failed: %w", err)
		}
	}(errs)

	wg.Add(1)
	go func() {
		defer l.Info().Msg("Server has been shutdown")
		defer wg.Done()
		<-ctx.Done()

		shutDownTimeoutCtx, cancelShutdownTimeCancel := context.WithTimeout(ctx, timeoutShutdown)
		defer cancelShutdownTimeCancel()
		if err := srv.Shutdown(shutDownTimeoutCtx); err != nil {
			l.Error().Err(err).Msg("an error occurred during server shutdown")
		}
	}()
}
