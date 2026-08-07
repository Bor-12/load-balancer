package proxy

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
}

func New(targetURL string, logger *slog.Logger) (*Proxy, error) {
	target, err := parseTargetURL(targetURL)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.ErrorHandler = func(responseWriter http.ResponseWriter, request *http.Request, err error) {
		logger.Error("backend request failed", "error", err, "path", request.URL.Path)
		http.Error(responseWriter, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	return &Proxy{reverseProxy: reverseProxy}, nil
}

func (proxy *Proxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	proxy.reverseProxy.ServeHTTP(responseWriter, request)
}

func parseTargetURL(targetURL string) (*url.URL, error) {
	if targetURL == "" {
		return nil, errors.New("backend URL is required")
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, errors.New("backend URL scheme must be http or https")
	}

	if target.Host == "" {
		return nil, errors.New("backend URL host is required")
	}

	return target, nil
}
