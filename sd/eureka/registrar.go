package eureka

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

type registrar struct {
	client  Client
	service *Service
	logger  log.Logger
}

func NewRegistrar(service *Service, client Client, logger log.Logger) sd.Registrar {
	return &registrar{
		service: service,
		client:  client,
		logger:  logger,
	}
}

func (r *registrar) Register() {
	if err := r.client.Register(r.service); err != nil {
		r.logger.Log("err", err)
	} else {
		r.logger.Log("action", "register")
	}
}

func (r *registrar) Deregister() {
	if err := r.client.Deregister(r.service); err != nil {
		r.logger.Log("err", err)
	} else {
		r.logger.Log("action", "deregister")
	}
}
