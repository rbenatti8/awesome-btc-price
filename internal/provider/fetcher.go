package provider

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"log/slog"
	"time"
)

//go:generate go tool mockgen --source=fetcher.go --destination=./mocks_test.go --package provider --typed
type database interface {
	Add(item BTCPrice)
	Query(filterFn func(item BTCPrice) bool) []BTCPrice
}

type broadcaster interface {
	Broadcast(message []byte)
}

type btcPriceProvider interface {
	GetPrice(context.Context) (BTCPrice, error)
}

// BTCPriceFetcher handles fetching BTC prices, storing them, and broadcasting updates.
// It interacts with a database, a BTC price provider, and a broadcaster.
type BTCPriceFetcher struct {
	store     database
	api       btcPriceProvider
	broadcast broadcaster
}

// NewFetcher initializes and returns a new instance of BTCPriceFetcher with the provided database, API, and broadcaster.
func NewFetcher(store database, api btcPriceProvider, bd broadcaster) *BTCPriceFetcher {
	return &BTCPriceFetcher{
		store:     store,
		api:       api,
		broadcast: bd,
	}
}

// Start begins the periodic polling of BTC prices at the specified interval until the context is canceled.
func (f *BTCPriceFetcher) Start(ctx context.Context, interval time.Duration) {
	go f.poll(ctx, interval)
}

func (f *BTCPriceFetcher) poll(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			price, err := f.api.GetPrice(ctx)
			if err != nil {
				slog.Error(fmt.Sprintf("error fetching price: %v", err))
				continue
			}

			f.store.Add(price)

			b, _ := json.Marshal(price)
			f.broadcast.Broadcast(b)
		}
	}
}
