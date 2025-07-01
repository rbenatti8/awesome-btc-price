package web

import (
	"context"
	fasthttpWebsocket "github.com/fasthttp/websocket"
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rbenatti8/awesome-btc-price/internal/provider"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"log"
	"net"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	db := NewMockdatabase(gomock.NewController(t))
	hub := NewHub()

	handler := NewHandler(db, hub)
	assert.NotNil(t, handler)
}

func TestHandler_Handle(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		db            func(ctrl *gomock.Controller) *Mockdatabase
		setupProvider func(hub *Hub)
		runAsserts    func(t *testing.T, result []byte, err error)
	}{
		{
			name: "Valid request with since parameter",
			url:  "ws://localhost:3001/ws?since=12345",
			db: func(ctrl *gomock.Controller) *Mockdatabase {
				db := NewMockdatabase(ctrl)
				db.EXPECT().Query(gomock.Any()).Return([]provider.BTCPrice{
					{
						Timestamp: 12345,
						PriceUSD:  decimal.NewFromFloat(12323.23),
					},
				})

				return db
			},
			runAsserts: func(t *testing.T, result []byte, err error) {
				assert.NoError(t, err)
				var btcPrice provider.BTCPrice

				err = json.Unmarshal(result, &btcPrice)
				assert.NoError(t, err)
				assert.Equal(t, decimal.NewFromFloat(12323.23).String(), btcPrice.PriceUSD.String())
				assert.Equal(t, int64(12345), btcPrice.Timestamp)
			},
		},
		{
			name: "Valid request without since parameter",
			url:  "ws://localhost:3001/ws",
			db: func(ctrl *gomock.Controller) *Mockdatabase {
				return NewMockdatabase(ctrl)
			},
			setupProvider: func(hub *Hub) {
				go func() {
					for {
						b, _ := json.Marshal(provider.BTCPrice{
							Timestamp: 12345,
							PriceUSD:  decimal.NewFromFloat(12323.23),
						})

						hub.Broadcast(b)
						time.Sleep(1 * time.Second)
					}
				}()
			},
			runAsserts: func(t *testing.T, result []byte, err error) {
				assert.NoError(t, err)
				var priceUSD provider.BTCPrice
				err = json.Unmarshal(result, &priceUSD)
				assert.NoError(t, err)
				assert.Equal(t, decimal.NewFromFloat(12323.23), priceUSD.PriceUSD)
				assert.Equal(t, int64(12345), priceUSD.Timestamp)
			},
		},
		{
			name: "Invalid request with since parameter",
			url:  "ws://localhost:3001/ws?since=invalid",
			db: func(ctrl *gomock.Controller) *Mockdatabase {
				return NewMockdatabase(ctrl)
			},
			setupProvider: func(hub *Hub) {
				go func() {
					for {
						b, _ := json.Marshal(provider.BTCPrice{
							Timestamp: 12345,
							PriceUSD:  decimal.NewFromFloat(12323.23),
						})

						hub.Broadcast(b)
						time.Sleep(1 * time.Second)
					}
				}()
			},
			runAsserts: func(t *testing.T, result []byte, err error) {
				assert.NoError(t, err)
				var msg = map[string]any{}
				err = json.Unmarshal(result, &msg)
				assert.NoError(t, err)

				assert.Equal(t, true, msg["error"])
				assert.Equal(t, "Invalid parameters provided", msg["message"])
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl, ctx := gomock.WithContext(ctx, t)
			defer ctrl.Finish()

			db := test.db(ctrl)
			hub := NewHub()
			go hub.Run(ctx)

			h := NewHandler(db, hub)

			app := setupTestApp(h.Handle)
			defer app.Shutdown()

			conn, _, err := fasthttpWebsocket.DefaultDialer.Dial(test.url, nil)
			require.NoError(t, err)
			assert.NotNil(t, conn)
			defer conn.Close()

			if test.setupProvider != nil {
				test.setupProvider(hub)
			}

			_, b, err := conn.ReadMessage()
			test.runAsserts(t, b, err)
		})
	}

}

func setupTestApp(h func(c *websocket.Conn)) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("ctx", c.Context())
			return c.Next()
		}

		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(h))

	go func() {
		if err := app.Listen(":3001"); err != nil {
			log.Fatal(err)
		}
	}()

	readyCh := make(chan struct{})

	go func() {
		for {
			conn, err := net.Dial("tcp", "localhost:3001")
			if err != nil {
				continue
			}

			if conn != nil {
				readyCh <- struct{}{}
				conn.Close()
				break
			}
		}
	}()

	<-readyCh

	return app
}
