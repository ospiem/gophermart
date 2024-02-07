package restclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ospiem/gophermart/internal/config"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/rs/zerolog"
)

var ErrOrderNotRegister = errors.New("the order does not registered")
var ErrTooManyRequests = errors.New("too many requests")

const DelayTime = "delayTime"

type Storage interface {
	ProcessOrderWithBonuses(ctx context.Context, orders models.Order, l *zerolog.Logger) error
	SelectOrdersToProceed(ctx context.Context, pagination int, offset *int) ([]models.Order, error)
}
type RestClient struct {
	Storage Storage
	Logger  *zerolog.Logger
	Cfg     *config.Config
}

func New(cfg *config.Config, s Storage, l *zerolog.Logger) *RestClient {
	return &RestClient{
		Storage: s,
		Logger:  l,
		Cfg:     cfg,
	}
}

func (r *RestClient) Run(ctx context.Context, wg *sync.WaitGroup) {
	mu := &sync.RWMutex{}
	delayMap := make(map[string]int, 1)
	orderCh := make(chan models.Order, r.Cfg.Pagination*r.Cfg.WorkersNum)

	for i := 0; i < r.Cfg.WorkersNum; i++ {
		wg.Add(1)
		go r.ProcessOrder(ctx, wg, mu, delayMap, orderCh)
		r.Logger.Debug().Msgf("Started worker #%d", i+1)
	}

	wg.Add(1)
	// Connection manager
	go func() {
		defer wg.Done() // Decrement the WaitGroup when the goroutine exits
		offset := 0
		for {
			select {
			case <-ctx.Done():
				r.Logger.Info().Msg("Stopped connection manager")
				return

			default:
				// Read delay from delayMap using RLock
				mu.RLock()
				delay := delayMap[DelayTime]
				mu.RUnlock()

				if delay != 0 {
					// If delay is not zero, sleep and reset delay in delayMap using Lock
					mu.Lock()
					time.Sleep(time.Duration(delay) * time.Second)
					delayMap[DelayTime] = 0
					mu.Unlock()
				}

				// Fetch orders from storage
				orders, err := r.Storage.SelectOrdersToProceed(ctx, r.Cfg.Pagination, &offset)
				if err != nil {
					r.Logger.Err(err)
				}
				// Send orders to orderCh
				for _, o := range orders {
					orderCh <- o
				}
				offset += len(orders)
			}
		}
	}()
}

func (r *RestClient) ProcessOrder(ctx context.Context, wg *sync.WaitGroup, mu *sync.RWMutex,
	delayMap map[string]int, jobs chan models.Order) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			r.Logger.Info().Msg("Stopped worker")
			return

		case order := <-jobs:
			r.Logger.Debug().Msgf("got new order %s", order.ID)
			updatedOrder, err := r.getOrderStatusFromService(ctx, order, mu, delayMap)
			if err != nil {
				if !errors.Is(err, ErrOrderNotRegister) && !errors.Is(err, ErrTooManyRequests) {
					r.Logger.Err(err).Msg("cannot get order status from accrual")
					continue
				}
			}
			if err := r.Storage.ProcessOrderWithBonuses(ctx, updatedOrder, r.Logger); err != nil {
				r.Logger.Err(err)
			}
		}
	}
}

func (r *RestClient) getOrderStatusFromService(ctx context.Context, order models.Order, mu *sync.RWMutex,
	delayMap map[string]int) (models.Order, error) {
	// Read from the map to check if the accrual service is available for establishing connections.
	mu.RLock()
	_ = delayMap[DelayTime]
	mu.RUnlock()
	client := http.Client{}

	apiURL := fmt.Sprintf("%v/api/orders/%v", r.Cfg.AccrualSysAddress, order.ID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return models.Order{}, fmt.Errorf("cannot generate request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")

	resp, err := client.Do(request)
	if err != nil {
		r.Logger.Err(err).Msg("cannot proceed request to accrual")
	}
	defer func() {
		if resp != nil {
			if err := resp.Body.Close(); err != nil {
				r.Logger.Err(err).Msg("cannot close body")
			}
		}
	}()

	if resp != nil {
		if resp.StatusCode == http.StatusOK {
			order := models.Order{}
			if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
				return models.Order{}, fmt.Errorf("cannot decode response: %w", err)
			}
			r.Logger.Info().Any("order", order).Msg("")
			return order, nil
		}

		if resp.StatusCode == http.StatusNoContent {
			return models.Order{}, ErrOrderNotRegister
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			delay, err := strconv.Atoi(resp.Header.Get("Retry-After"))
			if err != nil {
				return models.Order{}, fmt.Errorf("cannot get delay time: %w", err)
			}
			// Write to the map to pause all workers for the specified delay
			mu.Lock()
			delayMap[DelayTime] = delay
			mu.Unlock()
			return models.Order{}, ErrTooManyRequests
		}
		return models.Order{}, fmt.Errorf("unknown status code %d", resp.StatusCode)
	}
	return models.Order{}, errors.New("nil response")
}
