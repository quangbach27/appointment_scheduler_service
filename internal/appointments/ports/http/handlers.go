package http

import (
	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

type Handlers struct {
	service *app.Service
}

func NewHandlers(service *app.Service) Handlers {
	if service == nil {
		panic("service can't be nil")
	}

	return Handlers{
		service: service,
	}
}

func Register(router common.EchoRouter, handlers Handlers) error {
	RegisterHandlers(router, NewStrictHandler(handlers, nil))
	return nil
}
