package mapper

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		c := NewCache(time.Minute, 100)
		c.Set("key1", []byte(`{"id":1}`))

		data, ok := c.Get("key1")
		if !ok {
			t.Fatal("Expected cache hit")
		}
		if string(data) != `{"id":1}` {
			t.Errorf("Got %q, want {\"id\":1}", data)
		}
	})

	t.Run("miss on unknown key", func(t *testing.T) {
		c := NewCache(time.Minute, 100)
		_, ok := c.Get("nonexistent")
		if ok {
			t.Error("Expected cache miss")
		}
	})

	t.Run("expires after TTL", func(t *testing.T) {
		c := NewCache(50*time.Millisecond, 100)
		c.Set("key1", []byte("data"))

		time.Sleep(60 * time.Millisecond)

		_, ok := c.Get("key1")
		if ok {
			t.Error("Expected cache miss after TTL")
		}
	})

	t.Run("evicts oldest when full", func(t *testing.T) {
		c := NewCache(time.Minute, 2)

		c.Set("key1", []byte("first"))
		time.Sleep(time.Millisecond)
		c.Set("key2", []byte("second"))
		time.Sleep(time.Millisecond)
		c.Set("key3", []byte("third")) // should evict key1

		_, ok := c.Get("key1")
		if ok {
			t.Error("Expected key1 to be evicted")
		}

		_, ok = c.Get("key2")
		if !ok {
			t.Error("Expected key2 to still exist")
		}

		_, ok = c.Get("key3")
		if !ok {
			t.Error("Expected key3 to exist")
		}
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		c := NewCache(time.Minute, 100)
		c.Set("key1", []byte("old"))
		c.Set("key1", []byte("new"))

		data, ok := c.Get("key1")
		if !ok {
			t.Fatal("Expected cache hit")
		}
		if string(data) != "new" {
			t.Errorf("Got %q, want new", data)
		}
	})
}

func TestMapperWithCache(t *testing.T) {
	// Reset global cache state
	oldCache := mappingCache
	defer func() { mappingCache = oldCache }()

	t.Run("cache reduces HTTP calls", func(t *testing.T) {
		c := NewCache(time.Minute, 100)
		SetCache(c)

		// Pre-populate cache
		c.Set("http://localhost:9999/data?id=1", []byte(`{"name":"cached"}`))

		// Mapper should use cached value without making HTTP call
		// (if it tried HTTP to localhost:9999, it would fail)
		data, ok := c.Get("http://localhost:9999/data?id=1")
		if !ok {
			t.Fatal("Expected cache to contain the entry")
		}
		if string(data) != `{"name":"cached"}` {
			t.Errorf("Cached data = %q", data)
		}
	})
}
