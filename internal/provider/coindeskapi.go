package provider

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/shopspring/decimal"
	"resty.dev/v3"
	"time"
)

type BTCPrice struct {
	PriceUSD  decimal.Decimal
	Timestamp int64
}

type Config struct {
	tick time.Duration
}

type CoinDesk struct {
	httpClient *resty.Client
	clock      func() time.Time
}

func NewCoinDesk(httpClient *resty.Client) *CoinDesk {
	return &CoinDesk{
		httpClient: httpClient,
		clock:      time.Now,
	}
}

func (c *CoinDesk) GetPrice(ctx context.Context) (BTCPrice, error) {
	url := "/data/price?fsym=BTC&tsyms=USD"
	res, err := c.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return BTCPrice{}, fmt.Errorf("couldn't determine btc price: %v", err)
	}

	if res.IsError() {
		return BTCPrice{}, fmt.Errorf("unexpeected status code: %v", res.StatusCode())
	}

	var dto struct {
		PriceUSD decimal.Decimal `json:"USD"`
	}

	if err = json.Unmarshal(res.Bytes(), &dto); err != nil {
		return BTCPrice{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return BTCPrice{PriceUSD: dto.PriceUSD, Timestamp: c.clock().UTC().UnixNano()}, nil
}
