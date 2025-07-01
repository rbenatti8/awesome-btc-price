package database

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewInMemoryDB(t *testing.T) {
	t.Parallel()

	t.Run("TestNewInMemoryDB", func(t *testing.T) {
		t.Parallel()
		db := NewInMemoryDB[int]()
		defer db.Stop()

		assert.NotNil(t, db)
		assert.NotNil(t, db.ctx)
		assert.Equal(t, 10*time.Minute, db.cfg.ttl)
		assert.Equal(t, 1024, db.cfg.size)
		assert.Equal(t, 10*time.Minute, db.cfg.interval)
	})

	t.Run("TestNewInMemoryDB WithSize", func(t *testing.T) {
		t.Parallel()
		db := NewInMemoryDB[int](WithSize(2048))
		defer db.Stop()

		assert.Equal(t, 2048, db.cfg.size)
	})

	t.Run("TestNewInMemoryDB WithInterval", func(t *testing.T) {
		t.Parallel()
		db := NewInMemoryDB[int](WithInterval(1 * time.Millisecond))
		defer db.Stop()

		assert.Equal(t, 1*time.Millisecond, db.cfg.interval)
	})

	t.Run("TestNewInMemoryDB WithTTL", func(t *testing.T) {
		t.Parallel()
		db := NewInMemoryDB[int](WithTTL(1 * time.Minute))
		defer db.Stop()

		assert.Equal(t, 1*time.Minute, db.cfg.ttl)
	})

	t.Run("TestNewInMemoryDB WithClock", func(t *testing.T) {
		t.Parallel()

		customFunc := func() time.Time {
			return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
		}

		db := NewInMemoryDB[int](WithClock(customFunc))
		defer db.Stop()

		assert.NotNil(t, db)
		assert.Equal(t, time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC), db.cfg.clock())
	})
}

func TestInMemoryDB_Add(t *testing.T) {
	t.Parallel()

	db := NewInMemoryDB[int]()
	defer db.Stop()

	db.Add(1)
	assert.Len(t, db, 1)
	assert.Equal(t, 1, db.items[0].data)
}

func TestInMemoryDB_Query(t *testing.T) {
	t.Parallel()

	t.Run("Query items", func(t *testing.T) {
		t.Parallel()
		db := NewInMemoryDB[int]()
		defer db.Stop()

		db.Add(1)
		items := db.Query(nil)
		assert.Len(t, items, 1)
		assert.Equal(t, 1, items[0])
	})

	t.Run("Query on expired items", func(t *testing.T) {
		t.Parallel()
		db := NewInMemoryDB[int](WithTTL(800*time.Millisecond), WithInterval(800*time.Millisecond))
		defer db.Stop()

		db.Add(1)
		time.Sleep(1 * time.Second)
		db.Add(2)
		db.Add(3)
		items := db.Query(func(i int) bool {
			return i >= 2
		})

		assert.Len(t, items, 2)
		assert.Equal(t, 2, items[0])
		assert.Equal(t, 3, items[1])
	})
}
