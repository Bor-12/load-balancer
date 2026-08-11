package status

import (
	"net/http"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func HealthzHandler() http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	})
}

func ReadyzHandler(backends []*backend.Backend) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		for _, checkedBackend := range backends {
			if checkedBackend.IsAlive() {
				responseWriter.WriteHeader(http.StatusOK)
				return
			}
		}

		http.Error(responseWriter, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	})
}
