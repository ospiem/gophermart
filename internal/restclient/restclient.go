package restclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/ospiem/gophermart/internal/config"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/rs/zerolog"
)

var ErrOrderNotRegister = errors.New("the order does not registered")

const DelayTime = "delayTime"

type Storage interface {
	UpdateOrder(ctx context.Context, orders models.Order, l *zerolog.Logger) error
	SelectOrdersToProceed(ctx context.Context, offset int) ([]models.Order, error)
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

func (r *RestClient) GetOrders(ctx context.Context, wg *sync.WaitGroup, jobs chan models.Order) {
	wg.Add(1)
	offset := 0

	go func() {
		defer wg.Done()
		for {
			orders, err := r.Storage.SelectOrdersToProceed(ctx, offset)
			if err != nil {
				r.Logger.Err(err)
			}

			if len(orders) == 0 {
				continue
			}

			for _, o := range orders {
				jobs <- o
			}
			offset += len(orders)
		}
	}()
}
func (r *RestClient) ProcessOrder(ctx context.Context, wg *sync.WaitGroup, mu *sync.RWMutex,
	delayMap map[string]int, jobs chan models.Order) {

	defer wg.Done()

	for order := range jobs {
		updatedOrder, err := r.getOrderStatusFromService(ctx, order, mu, delayMap)
		if err != nil {
			r.Logger.Err(err)

			if err := r.Storage.UpdateOrder(ctx, updatedOrder, r.Logger); err != nil {
				r.Logger.Err(err)
			}
		}
	}
}

func (r *RestClient) getOrderStatusFromService(ctx context.Context, toProceed models.Order, mu *sync.RWMutex,
	delayMap map[string]int) (models.Order, error) {

	// Read from the map to check if the accrual service is available for establishing connections.
	mu.RLock()
	_ = delayMap[DelayTime]
	mu.RUnlock()

	client := http.Client{}

	apiURL := fmt.Sprintf("http://", r.Cfg.AccrualSysAddress, "/api/orders/", toProceed.ID)
	resp, err := client.Get(apiURL)
	if err != nil {
		r.Logger.Err(err).Msg("cannot proceed request to accrual")
	}

	if resp.StatusCode == http.StatusOK {
		order := models.Order{}
		if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
			return models.Order{}, fmt.Errorf("cannot decode response: %w", err)
		}
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
	}

	return models.Order{}, errors.New("unknown status code")
}
