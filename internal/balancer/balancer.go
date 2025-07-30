package balancer

import (
	"errors"
)

// Balancer interface for all load balancers
type Balancer interface {
	NextBackend() (*Backend, error)
}

func NewBalancer(method string, backends []*Backend) (Balancer, error) {
	switch method {
	case "roundrobin":
		return NewRoundRobinBalancer(backends), nil
	case "leastconn":
		return NewLeastConnBalancer(backends), nil
	case "weighted":
		return NewWeightedResponseTimeBalancer(backends), nil
	default:
		return nil, errors.New("Invalid balancer method: " + method)
	}
}
