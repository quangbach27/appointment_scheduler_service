package http

import (
	"context"

	"scheduler/internal/appointments/app"
	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
)

type ReservationReadModel interface {
	GetReservationByID(ctx context.Context, userID string, reservationUUID domain.ReservationUUID) (GetReservationResponse, error)
}

type Handlers struct {
	service              *app.Service
	reservationReadModel ReservationReadModel
}

func NewHandlers(service *app.Service, reservationReadModel ReservationReadModel) Handlers {
	if service == nil {
		panic("service can't be nil")
	}

	if reservationReadModel == nil {
		panic("reservationReadModel can't be nil")
	}

	return Handlers{
		service:              service,
		reservationReadModel: reservationReadModel,
	}
}

func Register(router common.EchoRouter, handlers Handlers) error {
	RegisterHandlers(router, NewStrictHandler(handlers, nil))
	return nil
}
