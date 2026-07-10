package middleware

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gflydev/cache"
)

// mockCache is an in-memory ICache implementation for tests. It honours TTLs and
// can be told to fail Set/Get to exercise the fail-open paths.
type mockCache struct {
	mu      sync.Mutex
	entries map[string]mockEntry
	failSet bool
	failGet bool
}

type mockEntry struct {
	value  interface{}
	expiry time.Time
}

func newMockCache() *mockCache {
	return &mockCache{entries: make(map[string]mockEntry)}
}

func (m *mockCache) Set(key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failSet {
		return errors.New("set failed")
	}
	m.entries[key] = mockEntry{value: value, expiry: time.Now().Add(expiration)}
	return nil
}

func (m *mockCache) Get(key string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failGet {
		return nil, errors.New("get failed")
	}
	e, ok := m.entries[key]
	if !ok {
		return nil, errors.New("not found")
	}
	if time.Now().After(e.expiry) {
		delete(m.entries, key)
		return nil, errors.New("expired")
	}
	return e.value, nil
}

func (m *mockCache) Del(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
	return nil
}

// setupCache registers a fresh mock cache and resets the keyed-mutex map so tests
// do not leak state into one another.
func setupCache(t *testing.T) *mockCache {
	t.Helper()
	mc := newMockCache()
	cache.Register(mc)
	keyedMutex = sync.Map{}
	return mc
}

func TestCheckRateLimitAllowsUpToLimit(t *testing.T) {
	setupCache(t)

	const max = 3
	for i := 1; i <= max; i++ {
		res := checkRateLimit("1.2.3.4:/api", max, 15)
		if res.limited {
			t.Fatalf("request %d was limited, want allowed", i)
		}
		if want := max - i; res.remaining != want {
			t.Errorf("request %d remaining = %d, want %d", i, res.remaining, want)
		}
	}
}

func TestCheckRateLimitBlocksOverLimit(t *testing.T) {
	setupCache(t)

	const max = 2
	checkRateLimit("ip:/p", max, 15)
	checkRateLimit("ip:/p", max, 15)

	res := checkRateLimit("ip:/p", max, 15)
	if !res.limited {
		t.Fatal("third request was allowed, want limited")
	}
	if res.remaining != 0 {
		t.Errorf("remaining = %d, want 0", res.remaining)
	}
	if res.retryAfter <= 0 {
		t.Errorf("retryAfter = %d, want positive", res.retryAfter)
	}
}

func TestCheckRateLimitSeparateKeys(t *testing.T) {
	setupCache(t)

	const max = 1
	if res := checkRateLimit("a:/p", max, 15); res.limited {
		t.Fatal("first key blocked unexpectedly")
	}
	// A different key must have its own independent counter.
	if res := checkRateLimit("b:/p", max, 15); res.limited {
		t.Fatal("second key blocked, counters are not independent")
	}
}

func TestCheckRateLimitWindowDoesNotResetTTL(t *testing.T) {
	mc := setupCache(t)

	const max = 5
	checkRateLimit("ip:/p", max, 15)
	firstExpiry := mc.entries["rate_limit:ip:/p"].expiry

	// A short pause then another request: the window end must not move forward.
	time.Sleep(10 * time.Millisecond)
	checkRateLimit("ip:/p", max, 15)
	secondExpiry := mc.entries["rate_limit:ip:/p"].expiry

	if secondExpiry.After(firstExpiry.Add(time.Millisecond)) {
		t.Errorf("window expiry advanced on increment: first=%v second=%v", firstExpiry, secondExpiry)
	}
}

func TestCheckRateLimitFailOpenOnCacheError(t *testing.T) {
	mc := setupCache(t)
	mc.failSet = true

	// Even though the counter can never be written, requests must be allowed.
	res := checkRateLimit("ip:/p", 1, 15)
	if res.limited {
		t.Fatal("request limited despite cache failure, want fail-open")
	}
}

func TestCheckRateLimitConcurrentDoesNotExceedLimit(t *testing.T) {
	setupCache(t)

	const max = 50
	const workers = 200

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res := checkRateLimit("ip:/p", max, 15); !res.limited {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != max {
		t.Errorf("allowed %d requests, want exactly %d (race in counter)", allowed, max)
	}
}

func TestDecodeEntry(t *testing.T) {
	now := time.Unix(0, 1234567890)

	count, expiry, ok := decodeEntry(encodeEntry(7, now))
	if !ok {
		t.Fatal("decodeEntry failed on valid input")
	}
	if count != 7 {
		t.Errorf("count = %d, want 7", count)
	}
	if !expiry.Equal(now) {
		t.Errorf("expiry = %v, want %v", expiry, now)
	}

	// Legacy count-only values and non-strings must be rejected.
	if _, _, ok := decodeEntry("5"); ok {
		t.Error("decodeEntry accepted legacy count-only value")
	}
	if _, _, ok := decodeEntry(42); ok {
		t.Error("decodeEntry accepted non-string value")
	}
}
