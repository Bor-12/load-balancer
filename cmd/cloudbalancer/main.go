package main

import (
	"log/slog"

	"github.com/Bor-12/load-balancer/internal/app"
)

func main() {
	slog.Info(app.StartupMessage())
}
