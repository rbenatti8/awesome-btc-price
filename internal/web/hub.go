package web

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"sync"
	"unsafe"
)

const shardCount = 32

type Hub struct {
	shards    []*shard
	broadcast chan []byte
}

type shard struct {
	connections map[*websocket.Conn]struct{}
	mu          sync.RWMutex
}

func NewHub() *Hub {
	shards := make([]*shard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &shard{
			connections: make(map[*websocket.Conn]struct{}),
		}
	}

	return &Hub{
		shards:    shards,
		broadcast: make(chan []byte),
	}
}

func (h *Hub) getShard(conn *websocket.Conn) *shard {
	return h.shards[uint64(uintptr(unsafe.Pointer(conn)))%uint64(shardCount)]
}

// Register adds a new websocket connection to the appropriate shard within the hub, ensuring thread-safe access.
func (h *Hub) Register(conn *websocket.Conn) {
	sh := h.getShard(conn)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.connections[conn] = struct{}{}
}

// Unregister removes a websocket connection from the appropriate shard in the hub, ensuring thread-safe access.
func (h *Hub) Unregister(conn *websocket.Conn) {
	sh := h.getShard(conn)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.connections, conn)
}

// Len calculates and returns the total number of active websocket connections across all shards in the hub.
func (h *Hub) Len() int {
	total := 0

	for _, sh := range h.shards {
		sh.mu.RLock()
		total += len(sh.connections)
		sh.mu.RUnlock()
	}

	return total
}

// Broadcast sends the provided message to all active websocket connections managed by the hub.
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// Run starts the main loop of the Hub, handling context cancellation and broadcasting messages to all connections.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-h.broadcast:
			h.broadcastMsg(msg)
		}
	}
}

func (h *Hub) broadcastMsg(msg []byte) {
	var wg sync.WaitGroup

	for _, sh := range h.shards {
		wg.Add(1)
		go func(sh *shard) {
			defer wg.Done()
			h.processShard(sh, msg)
		}(sh)
	}

	wg.Wait()
}

func (h *Hub) processShard(sh *shard, msg []byte) {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	for conn := range sh.connections {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			h.Unregister(conn)
		}
	}
}
