package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Limiter struct {
	mu              sync.Mutex
	buckets         map[string]*bucket
	requestsPerSec  float64
	burst           float64
	clock           clock
	cleanupInterval time.Duration
}

type bucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

const defaultCleanupInterval = time.Minute

func New(requestsPerSecond float64, burst int) *Limiter {
	return newWithClock(requestsPerSecond, burst, realClock{})
}

func newWithClock(requestsPerSecond float64, burst int, limiterClock clock) *Limiter {
	return &Limiter{
		buckets:         make(map[string]*bucket),
		requestsPerSec:  requestsPerSecond,
		burst:           float64(burst),
		clock:           limiterClock,
		cleanupInterval: defaultCleanupInterval,
	}
}

func (limiter *Limiter) Allow(clientID string) bool {
	now := limiter.clock.Now()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	clientBucket, exists := limiter.buckets[clientID]
	if !exists {
		clientBucket = &bucket{
			tokens:   limiter.burst,
			updated:  now,
			lastSeen: now,
		}
		limiter.buckets[clientID] = clientBucket
	}

	elapsed := now.Sub(clientBucket.updated).Seconds()
	clientBucket.tokens = min(limiter.burst, clientBucket.tokens+elapsed*limiter.requestsPerSec)
	clientBucket.updated = now
	clientBucket.lastSeen = now

	if clientBucket.tokens < 1 {
		return false
	}

	clientBucket.tokens--
	return true
}

func (limiter *Limiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		clientID := clientIdentifier(request)
		if !limiter.Allow(clientID) {
			responseWriter.Header().Set("Retry-After", "1")
			http.Error(responseWriter, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(responseWriter, request)
	})
}

func (limiter *Limiter) Cleanup() {
	now := limiter.clock.Now()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	for clientID, clientBucket := range limiter.buckets {
		if now.Sub(clientBucket.lastSeen) > limiter.cleanupInterval {
			delete(limiter.buckets, clientID)
		}
	}
}

func clientIdentifier(request *http.Request) string {
	forwardedFor := request.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		firstValue := strings.Split(forwardedFor, ",")[0]
		return strings.TrimSpace(firstValue)
	}

	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}

	return request.RemoteAddr
}
