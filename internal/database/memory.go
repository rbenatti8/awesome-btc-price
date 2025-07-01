package database

import (
	"context"
	"sync"
	"time"
)

type Item[T any] struct {
	expiration int64
	data       T
}

type Config struct {
	size     int
	ttl      time.Duration
	interval time.Duration
	clock    func() time.Time
}

type InMemoryDB[T any] struct {
	mu    sync.RWMutex
	items []Item[T]
	cfg   *Config
	ctx   context.Context
}

type Option func(*Config)

func WithSize(size int) Option {
	return func(cfg *Config) {
		cfg.size = size
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.ttl = ttl
	}
}

func WithInterval(interval time.Duration) Option {
	return func(cfg *Config) {
		cfg.interval = interval
	}
}

func WithClock(clock func() time.Time) Option {
	return func(cfg *Config) {
		cfg.clock = clock
	}
}

func NewInMemoryDB[T any](opts ...Option) *InMemoryDB[T] {
	cfg := &Config{
		size:     1024,
		ttl:      10 * time.Minute,
		interval: 10 * time.Minute,
		clock:    time.Now,
	}

	for _, o := range opts {
		o(cfg)
	}

	db := &InMemoryDB[T]{
		cfg:   cfg,
		items: make([]Item[T], 0, cfg.size),
		ctx:   context.Background(),
	}

	db.startCleanup()

	return db
}

// Add inserts an item of type T into the in-memory database with an associated expiration time.
func (db *InMemoryDB[T]) Add(item T) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.items = append(db.items, Item[T]{expiration: db.cfg.clock().UTC().Add(db.cfg.ttl).UnixNano(), data: item})
}

// Query retrieves all items from the database that match the provided filter function and have not expired.
// If filterFn is nil, all non-expired items are returned.
func (db *InMemoryDB[T]) Query(filterFn func(T) bool) []T {
	db.mu.RLock()
	defer db.mu.RUnlock()

	now := db.cfg.clock().UTC().UnixNano()

	var result []T

	for _, item := range db.items {
		if item.expiration < now {
			continue
		}

		if filterFn == nil || filterFn(item.data) {
			result = append(result, item.data)
		}
	}

	return result
}

// Stop terminates the in-memory database's context, signaling all associated routines to halt operations.
func (db *InMemoryDB[T]) Stop() {
	if db.ctx != nil {
		db.ctx.Done()
	}
}

func (db *InMemoryDB[T]) cleanup() {
	db.mu.Lock()
	defer db.mu.Unlock()

	now := db.cfg.clock().UTC().UnixNano()
	filtered := db.items[:0]

	for _, item := range db.items {
		if item.expiration > now {
			filtered = append(filtered, item)
		}
	}

	db.items = filtered
}

func (db *InMemoryDB[T]) startCleanup() {
	go func() {
		ticker := time.NewTicker(db.cfg.interval)
		defer ticker.Stop()

		for {
			select {
			case <-db.ctx.Done():
				return
			case <-ticker.C:
				db.cleanup()
			}
		}
	}()
}
