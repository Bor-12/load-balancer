package ratelimit

import (
	"strconv"
	"testing"
)

func BenchmarkLimiterAllowSameClient(b *testing.B) {
	limiter := New(1000000, 1000000)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		limiter.Allow("client-1")
	}
}

func BenchmarkLimiterAllowManyClients(b *testing.B) {
	limiter := New(1000000, 1000000)

	b.ReportAllocs()
	b.ResetTimer()

	for index := range b.N {
		limiter.Allow("client-" + strconv.Itoa(index%1000))
	}
}
