package proxy

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
)

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
	balancer     balancer.Balancer
	logger       *slog.Logger
}

func New(targetURL string, logger *slog.Logger) (*Proxy, error) {
	targetBackend, err := backend.New("backend-1", targetURL)
	if err != nil {
		return nil, err
	}

	roundRobin, err := balancer.NewRoundRobin([]*backend.Backend{targetBackend})
	if err != nil {
		return nil, err
	}

	return NewWithBalancer(roundRobin, logger), nil
}

func NewWithBalancer(requestBalancer balancer.Balancer, logger *slog.Logger) *Proxy {
	if logger == nil {
		logger = slog.Default()
	}

	reverseProxy := &httputil.ReverseProxy{
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			target := selectedBackendURL(proxyRequest.In.Context())
			proxyRequest.SetURL(target)
		},
	}
	reverseProxy.ErrorHandler = func(responseWriter http.ResponseWriter, request *http.Request, err error) {
		logger.Error("backend request failed", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	return &Proxy{
		reverseProxy: reverseProxy,
		balancer:     requestBalancer,
		logger:       logger,
	}
}

func (proxy *Proxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	selectedBackend, err := proxy.balancer.Next()
	if err != nil {
		proxy.logger.Error("backend selection failed", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	requestContext := context.WithValue(request.Context(), selectedBackendURLKey{}, selectedBackend.URL)
	proxy.reverseProxy.ServeHTTP(responseWriter, request.WithContext(requestContext))
}

type selectedBackendURLKey struct{}

func selectedBackendURL(context context.Context) *url.URL {
	return context.Value(selectedBackendURLKey{}).(*url.URL)
}
