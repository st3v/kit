package static

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/loadbalancer"
	"github.com/go-kit/kit/loadbalancer/fixed"
	"github.com/go-kit/kit/log"
)

// Publisher yields a set of static endpoints as produced by the passed factory.
type Publisher struct{ *fixed.Publisher }

// NewPublisher returns a static endpoint Publisher.
func NewPublisher(instances []string, factory loadbalancer.Factory, logger log.Logger) Publisher {
	logger = log.NewContext(logger).With("component", "Fixed Publisher")
	endpoints := []endpoint.Endpoint{}
	for _, instance := range instances {
		e, err := factory(instance)
		if err != nil {
			_ = logger.Log("instance", instance, "err", err)
			continue
		}
		endpoints = append(endpoints, e)
	}
	return Publisher{fixed.NewPublisher(endpoints)}
}
