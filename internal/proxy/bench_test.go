package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkProxyServeHTTP(b *testing.B) {
	backendServer := newBackendServer("backend-1")
	defer backendServer.Close()

	reverseProxy := newTestProxy(b, backendServer.URL)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		responseRecorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		reverseProxy.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusOK {
			b.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
		}
	}
}
