package http

import (
	"context"
	"strings"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

func (h Handlers) ConfirmAppointment(ctx context.Context, request ConfirmAppointmentRequestObject) (ConfirmAppointmentResponseObject, error) {
	userID := strings.TrimSpace(request.Params.XUserId)
	if userID == "" {
		return nil, common.NewUnauthorizedError("missing-user-id", "X-User-Id header is required")
	}

	appointmentUUID, err := h.service.ConfirmAppointment(ctx, app.ConfirmAppointment{
		UserID:          userID,
		ReservationUUID: request.ReservationUuid,
	})
	if err != nil {
		return nil, err
	}

	return ConfirmAppointment200JSONResponse{
		AppointmentUuid: appointmentUUID,
	}, nil
}
