package web

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	assert.NotNil(t, hub)
}

func TestHub_Register(t *testing.T) {
	hub := NewHub()
	go hub.Run(context.Background())

	conn := &websocket.Conn{}
	hub.Register(conn)
	assert.Equal(t, 1, hub.Len())
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run(context.Background())

	conn := &websocket.Conn{}
	hub.Register(conn)
	assert.Equal(t, 1, hub.Len())

	hub.Unregister(conn)
	assert.Equal(t, 0, hub.Len())
}
