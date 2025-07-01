package provider

import (
	"context"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"resty.dev/v3"
	"testing"
)

func TestNewCoinDesk(t *testing.T) {
	c := resty.New()
	api := NewCoinDesk(c)
	assert.NotNil(t, api)
	assert.Equal(t, c, api.httpClient)
}

func TestCoinDesk_GetPrice(t *testing.T) {
	type mock struct {
		response   string
		statusCode int
	}

	tests := []struct {
		name       string
		mock       mock
		runAsserts func(t *testing.T, price BTCPrice, err error)
	}{
		{
			name: "should return current price",
			mock: mock{
				response:   `{"USD": 5000.00}`,
				statusCode: http.StatusOK,
			},
			runAsserts: func(t *testing.T, price BTCPrice, err error) {
				assert.NoError(t, err)
				assert.True(t, price.PriceUSD.Equal(decimal.NewFromInt(5000)))
				assert.NotZero(t, price.Timestamp)
			},
		},
		{
			name: "should return error on non-200 status code",
			mock: mock{
				response:   `{"error": "internal server error"}`,
				statusCode: http.StatusInternalServerError,
			},
			runAsserts: func(t *testing.T, _ BTCPrice, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "unexpeected status code: 500")
			},
		},
		{
			name: "should return error on invalid json",
			mock: mock{
				response:   `{}.`,
				statusCode: http.StatusOK,
			},
			runAsserts: func(t *testing.T, _ BTCPrice, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to unmarshal response")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/data/price" && r.Method == "GET" {
					w.Header().Set("Conteenet-type", "application/json")
					w.WriteHeader(test.mock.statusCode)
					_, _ = w.Write([]byte(test.mock.response))
					return
				}
			}))

			defer ts.Close()

			c := resty.New()
			c.SetBaseURL(ts.URL)

			api := NewCoinDesk(c)

			price, err := api.GetBTCPrice(context.Background())
			test.runAsserts(t, price, err)
		})
	}
}
