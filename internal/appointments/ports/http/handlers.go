package http

import (
	"errors"
	"strings"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

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

// requireUserID validates the X-User-Id header, common to every handler. The
// public error stays generic (no mention of the header) so an unauthenticated
// caller learns nothing about the auth mechanism; the specific reason is
// attached as an internal error, which EchoErrorHandler logs but never
// exposes in the response body.
func requireUserID(xUserID *string) (string, error) {
	userID := strings.TrimSpace(common.SafeDeref(xUserID, ""))
	if userID == "" {
		return "", common.NewUnauthorizedError("not-authenticated", "user is not authenticated").
			WithInternalError(errors.New("X-User-Id header is missing"))
	}

	return userID, nil
}
