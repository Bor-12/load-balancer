package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	maxAttempts  int
}

const defaultRequestTimeout = 5 * time.Second
const defaultMaxAttempts = 2

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
	return NewWithBalancerWithTimeoutAndRetries(requestBalancer, logger, defaultRequestTimeout, defaultMaxAttempts)
}

func NewWithBalancerWithTimeout(requestBalancer balancer.Balancer, logger *slog.Logger, timeout time.Duration) *Proxy {
	return NewWithBalancerWithTimeoutAndRetries(requestBalancer, logger, timeout, defaultMaxAttempts)
}

func NewWithBalancerWithTimeoutAndRetries(requestBalancer balancer.Balancer, logger *slog.Logger, timeout time.Duration, maxAttempts int) *Proxy {
	if logger == nil {
		logger = slog.Default()
	}

	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
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
		maxAttempts:  maxAttempts,
	}
}

func (proxy *Proxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	requestBody, err := reusableRequestBody(request)
	if err != nil {
		proxy.logger.Error("failed to read request body", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	attempts := proxy.attemptsFor(request)
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptRequest := cloneRequestWithBody(request, requestBody)
		if attempt == attempts {
			proxy.serveAttempt(responseWriter, attemptRequest)
			return
		}

		buffer := newResponseBuffer()
		proxy.serveAttempt(buffer, attemptRequest)

		if !shouldRetry(buffer.statusCode) {
			buffer.WriteTo(responseWriter)
			return
		}

		proxy.logger.Warn("retrying backend request", "path", request.URL.Path, "method", request.Method, "attempt", attempt, "status", buffer.statusCode)
	}
}

func (proxy *Proxy) serveAttempt(responseWriter http.ResponseWriter, request *http.Request) {
	selectedBackend, err := proxy.balancer.Next()
	if err != nil {
		proxy.logger.Error("backend selection failed", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	timeoutContext, cancel := context.WithTimeout(request.Context(), proxy.timeout)
	defer cancel()

	selectedBackend.IncrementActive()
	defer selectedBackend.DecrementActive()

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

func (proxy *Proxy) attemptsFor(request *http.Request) int {
	if !isIdempotent(request.Method) {
		return 1
	}

	return proxy.maxAttempts
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func shouldRetry(statusCode int) bool {
	return statusCode == http.StatusBadGateway || statusCode == http.StatusGatewayTimeout
}

func reusableRequestBody(request *http.Request) ([]byte, error) {
	if request.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func cloneRequestWithBody(request *http.Request, body []byte) *http.Request {
	clonedRequest := request.Clone(request.Context())
	if body != nil {
		clonedRequest.Body = io.NopCloser(bytes.NewReader(body))
	}

	return clonedRequest
}

type responseBuffer struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (buffer *responseBuffer) Header() http.Header {
	return buffer.header
}

func (buffer *responseBuffer) WriteHeader(statusCode int) {
	buffer.statusCode = statusCode
}

func (buffer *responseBuffer) Write(body []byte) (int, error) {
	return buffer.body.Write(body)
}

func (buffer *responseBuffer) WriteTo(responseWriter http.ResponseWriter) {
	for key, values := range buffer.header {
		for _, value := range values {
			responseWriter.Header().Add(key, value)
		}
	}

	responseWriter.WriteHeader(buffer.statusCode)
	_, _ = responseWriter.Write(buffer.body.Bytes())
}
