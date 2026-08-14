package http

import (
	"context"

	"scheduler/internal/appointments/app"
)

func (h Handlers) CancelBooking(ctx context.Context, request CancelBookingRequestObject) (CancelBookingResponseObject, error) {
	userID, err := requireUserID(request.Params.XUserId)
	if err != nil {
		return nil, err
	}

	if err := h.service.CancelBooking(ctx, app.CancelBooking{
		UserID:          userID,
		AppointmentUUID: request.AppointmentUuid,
	}); err != nil {
		return nil, err
	}

	return CancelBooking204Response{}, nil
}
