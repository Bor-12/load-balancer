package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
)

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
	balancer     balancer.Balancer
	logger       *slog.Logger
	timeout      time.Duration
}

const defaultRequestTimeout = 5 * time.Second

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
	return NewWithBalancerWithTimeout(requestBalancer, logger, defaultRequestTimeout)
}

func NewWithBalancerWithTimeout(requestBalancer balancer.Balancer, logger *slog.Logger, timeout time.Duration) *Proxy {
	if logger == nil {
		logger = slog.Default()
	}

	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	reverseProxy := &httputil.ReverseProxy{
		Transport: newTransport(timeout),
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			target := selectedBackendURL(proxyRequest.In.Context())
			proxyRequest.SetURL(target)
		},
	}
	reverseProxy.ErrorHandler = func(responseWriter http.ResponseWriter, request *http.Request, err error) {
		if isTimeout(err) {
			logger.Error("backend request timed out", "error", err, "path", request.URL.Path)
			http.Error(responseWriter, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
			return
		}

		logger.Error("backend request failed", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	return &Proxy{
		reverseProxy: reverseProxy,
		balancer:     requestBalancer,
		logger:       logger,
		timeout:      timeout,
	}
}

func (proxy *Proxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	selectedBackend, err := proxy.balancer.Next()
	if err != nil {
		proxy.logger.Error("backend selection failed", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	timeoutContext, cancel := context.WithTimeout(request.Context(), proxy.timeout)
	defer cancel()

	requestContext := context.WithValue(timeoutContext, selectedBackendURLKey{}, selectedBackend.URL)
	proxy.reverseProxy.ServeHTTP(responseWriter, request.WithContext(requestContext))
}

type selectedBackendURLKey struct{}

func selectedBackendURL(context context.Context) *url.URL {
	return context.Value(selectedBackendURLKey{}).(*url.URL)
}

func newTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var timeoutError net.Error
	return errors.As(err, &timeoutError) && timeoutError.Timeout()
}
