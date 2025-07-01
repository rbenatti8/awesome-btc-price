package web

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/websocket"
	"github.com/rbenatti8/awesome-btc-price/internal/provider"
	"log/slog"
	"strconv"
)

//go:generate go tool mockgen --source=handler.go --destination=./mocks_test.go --package web --typed
type database interface {
	Query(filterFn func(item provider.BTCPrice) bool) []provider.BTCPrice
}

type Handler struct {
	db  database
	hub *Hub
}

type Params struct {
	Since int64
}

func NewHandler(db database, hub *Hub) *Handler {
	return &Handler{
		db:  db,
		hub: hub,
	}
}

func (h *Handler) Handle(c *websocket.Conn) {
	defer func() {
		if err := c.Close(); err != nil {
			slog.Error("failed to close connection", slog.String("error", err.Error()))
		}
	}()

	params, err := h.getParams(c)
	if err != nil {
		h.handleErr(c, err)
		return
	}

	if err = h.sendInitialData(c, params.Since); err != nil {
		h.handleErr(c, err)
		return
	}

	h.hub.Register(c)
	defer h.hub.Unregister(c)

	h.readMessages(c)
}

func (h *Handler) readMessages(c *websocket.Conn) {
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			slog.Error("failed to read message", slog.String("error", err.Error()))
			return
		}
	}
}

func (h *Handler) sendInitialData(c *websocket.Conn, since int64) error {
	if since == 0 {
		return nil
	}

	filterFn := func(item provider.BTCPrice) bool {
		return item.Timestamp >= since
	}

	items := h.db.Query(filterFn)

	for _, item := range items {
		b, _ := json.Marshal(item)
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			slog.Error("failed to write message", slog.String("error", err.Error()))
			return errors.Join(err, fmt.Errorf("failed to send item: %v", item))
		}
	}

	return nil
}

func (h *Handler) getParams(c *websocket.Conn) (*Params, error) {
	var params Params
	since := c.Query("since")
	parsedSince, err := strconv.ParseInt(since, 10, 64)
	if since != "" && err != nil {
		details := map[string]string{"query.since": "must be a valid integer"}

		return nil, invalidParamError{
			message: "Invalid parameters provided",
			details: details,
		}
	}

	params.Since = parsedSince

	return &params, nil
}

type detailer interface {
	Details() map[string]string
}

func (h *Handler) handleErr(c *websocket.Conn, err error) {
	if err == nil {
		return
	}

	response := map[string]any{
		"error":   true,
		"message": err.Error(),
	}

	var dErr detailer
	if errors.As(err, &dErr) {
		response["details"] = dErr.Details()
	}

	b, err := json.Marshal(response)

	if err = c.WriteMessage(websocket.TextMessage, b); err != nil {
		slog.Error("failed to write message", slog.String("error", err.Error()))
	}
}
