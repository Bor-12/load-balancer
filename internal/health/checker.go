package health

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
)

type Checker struct {
	backends []*backend.Backend
	client   *http.Client
	path     string
	interval time.Duration
	logger   *slog.Logger
	observer Observer
}

type Observer interface {
	UpdateBackendState(updatedBackend *backend.Backend)
}

type noopObserver struct{}

func (observer noopObserver) UpdateBackendState(updatedBackend *backend.Backend) {
}

func NewChecker(backends []*backend.Backend, path string, interval time.Duration, timeout time.Duration, logger *slog.Logger) *Checker {
	return NewCheckerWithObserver(backends, path, interval, timeout, logger, noopObserver{})
}

func NewCheckerWithObserver(backends []*backend.Backend, path string, interval time.Duration, timeout time.Duration, logger *slog.Logger, observer Observer) *Checker {
	if path == "" {
		path = "/health"
	}

	if interval <= 0 {
		interval = 2 * time.Second
	}

	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}

	if logger == nil {
		logger = slog.Default()
	}

	if observer == nil {
		observer = noopObserver{}
	}

	return &Checker{
		backends: backends,
		client: &http.Client{
			Timeout: timeout,
		},
		path:     path,
		interval: interval,
		logger:   logger,
		observer: observer,
	}
}

func (checker *Checker) Run(context context.Context) {
	checker.CheckOnce(context)

	ticker := time.NewTicker(checker.interval)
	defer ticker.Stop()

	for {
		select {
		case <-context.Done():
			return
		case <-ticker.C:
			checker.CheckOnce(context)
		}
	}
}

func (checker *Checker) CheckOnce(context context.Context) {
	for _, checkedBackend := range checker.backends {
		wasAlive := checkedBackend.IsAlive()
		alive := checker.isBackendAlive(context, checkedBackend)
		checkedBackend.SetAlive(alive)
		checker.observer.UpdateBackendState(checkedBackend)
		if wasAlive != alive {
			checker.logger.Info("backend health changed", "backend_id", checkedBackend.ID, "old_state", wasAlive, "new_state", alive)
		}
	}
}

func (checker *Checker) isBackendAlive(context context.Context, checkedBackend *backend.Backend) bool {
	healthURL := checkedBackend.URL.ResolveReference(&url.URL{Path: checker.path})
	request, err := http.NewRequestWithContext(context, http.MethodGet, healthURL.String(), nil)
	if err != nil {
		checker.logger.Error("failed to create health check request", "backend_id", checkedBackend.ID, "error", err)
		return false
	}

	response, err := checker.client.Do(request)
	if err != nil {
		checker.logger.Warn("backend health check failed", "backend_id", checkedBackend.ID, "error", err)
		return false
	}
	defer response.Body.Close()

	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}
