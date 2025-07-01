package provider

import (
	"context"
	"github.com/goccy/go-json"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestNewFetcher(t *testing.T) {
	ctrl := gomock.NewController(t)

	api := NewMockbtcPriceProvider(ctrl)
	store := NewMockdatabase(ctrl)
	bd := NewMockbroadcaster(ctrl)

	f := NewFetcher(store, api, bd)

	assert.NotNil(t, f)
}

func TestBTCPriceFetcher_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewMockbtcPriceProvider(ctrl)
	store := NewMockdatabase(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := time.Date(2025, 6, 30, 10, 20, 0, 0, time.UTC)
	expectedPrice := BTCPrice{PriceUSD: decimal.NewFromFloat(1000.15), Timestamp: d.UnixNano()}

	store.EXPECT().Add(expectedPrice).MinTimes(1)
	api.EXPECT().GetPrice(ctx).AnyTimes().Return(expectedPrice, nil).MinTimes(1)

	b, _ := json.Marshal(expectedPrice)

	bd := NewMockbroadcaster(ctrl)
	bd.EXPECT().Broadcast(b).MinTimes(1)

	f := NewFetcher(store, api, bd)
	f.Start(ctx, 100*time.Millisecond)

	time.Sleep(300 * time.Millisecond)
}
