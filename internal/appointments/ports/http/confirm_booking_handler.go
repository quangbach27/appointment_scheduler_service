package http

import (
	"context"
	"strings"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

func (h Handlers) ConfirmBooking(ctx context.Context, request ConfirmBookingRequestObject) (ConfirmBookingResponseObject, error) {
	userID := strings.TrimSpace(common.SafeDeref(request.Params.XUserId, ""))
	if userID == "" {
		return nil, common.NewUnauthorizedError("missing-user-id", "X-User-Id header is required")
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
