package http

import (
	"context"
	"strings"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

func (h Handlers) CancelAppointment(ctx context.Context, request CancelAppointmentRequestObject) (CancelAppointmentResponseObject, error) {
	userID := strings.TrimSpace(request.Params.XUserId)
	if userID == "" {
		return nil, common.NewUnauthorizedError("missing-user-id", "X-User-Id header is required")
	}

	if err := h.service.CancelAppointment(ctx, app.CancelAppointment{
		UserID:          userID,
		AppointmentUUID: request.AppointmentUuid,
	}); err != nil {
		return nil, err
	}

	return CancelAppointment204Response{}, nil
}
