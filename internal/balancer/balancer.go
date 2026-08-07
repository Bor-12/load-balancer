package balancer

import "github.com/Bor-12/load-balancer/internal/backend"

type Balancer interface {
	Next() (*backend.Backend, error)
}
