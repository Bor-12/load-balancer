package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiter_AllowsRequestsWithinBurst(t *testing.T) {
	testClock := newFakeClock()
	limiter := newWithClock(1, 3, testClock)

	for range 3 {
		if !limiter.Allow("client-1") {
			t.Fatal("expected request to be allowed")
		}
	}
}

func TestLimiter_RejectsRequestsWhenBucketIsEmpty(t *testing.T) {
	testClock := newFakeClock()
	limiter := newWithClock(1, 2, testClock)

	if !limiter.Allow("client-1") {
		t.Fatal("expected first request to be allowed")
	}
	if !limiter.Allow("client-1") {
		t.Fatal("expected second request to be allowed")
	}
	if limiter.Allow("client-1") {
		t.Fatal("expected third request to be rejected")
	}
}

func TestLimiter_RefillsTokensOverTime(t *testing.T) {
	testClock := newFakeClock()
	limiter := newWithClock(2, 2, testClock)

	if !limiter.Allow("client-1") {
		t.Fatal("expected first request to be allowed")
	}
	if !limiter.Allow("client-1") {
		t.Fatal("expected second request to be allowed")
	}

	testClock.Advance(500 * time.Millisecond)

	if !limiter.Allow("client-1") {
		t.Fatal("expected request to be allowed after refill")
	}
	if limiter.Allow("client-1") {
		t.Fatal("expected request to be rejected before another token is refilled")
	}
}

func TestLimiter_TracksClientsIndependently(t *testing.T) {
	testClock := newFakeClock()
	limiter := newWithClock(1, 1, testClock)

	if !limiter.Allow("client-1") {
		t.Fatal("expected client-1 request to be allowed")
	}
	if limiter.Allow("client-1") {
		t.Fatal("expected client-1 second request to be rejected")
	}
	if !limiter.Allow("client-2") {
		t.Fatal("expected client-2 request to be allowed")
	}
}

func TestLimiter_HandlerReturnsTooManyRequests(t *testing.T) {
	testClock := newFakeClock()
	limiter := newWithClock(1, 1, testClock)
	handler := limiter.Handler(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	firstRequest.RemoteAddr = "192.0.2.1:12345"
	handler.ServeHTTP(firstRecorder, firstRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, firstRecorder.Code)
	}

	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	secondRequest.RemoteAddr = "192.0.2.1:23456"
	handler.ServeHTTP(secondRecorder, secondRequest)

	if secondRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, secondRecorder.Code)
	}
}

func TestLimiter_UsesForwardedForHeader(t *testing.T) {
	testClock := newFakeClock()
	limiter := newWithClock(1, 1, testClock)

	firstRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	firstRequest.RemoteAddr = "192.0.2.1:12345"
	firstRequest.Header.Set("X-Forwarded-For", "203.0.113.10, 192.0.2.1")

	secondRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	secondRequest.RemoteAddr = "192.0.2.1:23456"
	secondRequest.Header.Set("X-Forwarded-For", "203.0.113.11, 192.0.2.1")

	if !limiter.Allow(clientIdentifier(firstRequest)) {
		t.Fatal("expected first forwarded client to be allowed")
	}
	if !limiter.Allow(clientIdentifier(secondRequest)) {
		t.Fatal("expected second forwarded client to be allowed")
	}
}

type fakeClock struct {
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
}

func (clock *fakeClock) Now() time.Time {
	return clock.now
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.now = clock.now.Add(duration)
}
