package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Runner struct {
	server          *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func NewRunner(server *http.Server, logger *slog.Logger, shutdownTimeout time.Duration) *Runner {
	if logger == nil {
		logger = slog.Default()
	}

	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	return &Runner{
		server:          server,
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
	}
}

func (runner *Runner) Run(context context.Context) error {
	serverErrors := make(chan error, 1)
	go func() {
		runner.logger.Info("server started", "address", runner.server.Addr)
		serverErrors <- runner.server.ListenAndServe()
	}()

	select {
	case <-context.Done():
		return runner.Shutdown()
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	}
}

func (runner *Runner) Shutdown() error {
	runner.logger.Info("shutdown initiated")

	shutdownContext, cancel := context.WithTimeout(context.Background(), runner.shutdownTimeout)
	defer cancel()

	err := runner.server.Shutdown(shutdownContext)
	if err != nil {
		return err
	}

	runner.logger.Info("shutdown completed")

	return nil
}
