package backend

import (
	"errors"
	"net/url"
	"sync"
)

type Backend struct {
	ID  string
	URL *url.URL

	mutex sync.RWMutex
	alive bool
}

func New(id string, rawURL string) (*Backend, error) {
	parsedURL, err := parseURL(rawURL)
	if err != nil {
		return nil, err
	}

	return &Backend{
		ID:    id,
		URL:   parsedURL,
		alive: true,
	}, nil
}

func (backend *Backend) IsAlive() bool {
	backend.mutex.RLock()
	defer backend.mutex.RUnlock()

	return backend.alive
}

func (backend *Backend) SetAlive(alive bool) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()

	backend.alive = alive
}

func parseURL(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return nil, errors.New("backend URL is required")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("backend URL scheme must be http or https")
	}

	if parsedURL.Host == "" {
		return nil, errors.New("backend URL host is required")
	}

	return parsedURL, nil
}
