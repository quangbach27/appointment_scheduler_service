package http

import (
	"context"

	"scheduler/internal/appointments/app"
)

func (h Handlers) ConfirmBooking(ctx context.Context, request ConfirmBookingRequestObject) (ConfirmBookingResponseObject, error) {
	userID, err := requireUserID(request.Params.XUserId)
	if err != nil {
		return nil, err
	}

	appointmentUUID, err := h.service.ConfirmBooking(ctx, app.ConfirmBooking{
		UserID:          userID,
		ReservationUUID: request.ReservationUuid,
	})
	if err != nil {
		return nil, err
	}

	return ConfirmBooking200JSONResponse{
		AppointmentUuid: appointmentUUID,
	}, nil
}
