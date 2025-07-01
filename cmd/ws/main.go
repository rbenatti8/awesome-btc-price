package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rbenatti8/awesome-btc-price/internal/database"
	"github.com/rbenatti8/awesome-btc-price/internal/provider"
	"github.com/rbenatti8/awesome-btc-price/internal/web"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"resty.dev/v3"
	"syscall"
	"time"
)

func main() {
	token := getEnv("TOKEN")
	pollingInterval := 5 * time.Second

	setupLimits()

	client := resty.New()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client.SetBaseURL("https://min-api.cryptocompare.com")
	client.SetHeader("Authorization", "Apikey "+token)

	client.SetTransport(transport)
	client.SetRetryCount(3)
	client.SetTimeout(pollingInterval)
	client.SetRetryWaitTime(2 * time.Second)
	client.SetRetryMaxWaitTime(10 * time.Second)

	defer func() {
		if err := client.Close(); err != nil {
			slog.Error(err.Error())
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := database.NewInMemoryDB[provider.BTCPrice]()
	api := provider.NewCoinDesk(client)
	hub := web.NewHub()
	fetcher := provider.NewFetcher(db, api, hub)
	handler := web.NewHandler(db, hub)

	go hub.Run(ctx)
	go fetcher.Start(ctx, pollingInterval)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	app.Use(recover.New())
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}

		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(handler.Handle))

	go func() {
		if err := app.Listen(":3000"); err != nil {
			slog.Error(err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

	slog.Info(fmt.Sprintf("signal %v received", <-quit), slog.Attr{})

	if err := app.Shutdown(); err != nil {
		slog.Error(fmt.Sprintf("shutdown: %v", err))
	}
}

func getEnv(key string) string {
	value, found := os.LookupEnv(key)
	if !found {
		log.Fatal("Environment variable not found: " + key)
	}

	return value
}

// REF https://github.com/eranyanay/1m-go-websockets/tree/master/2_ws_ulimit
func setupLimits() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Fatal(err)
	}

	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Fatal(err)
	}
}
