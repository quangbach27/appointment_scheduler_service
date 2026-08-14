package http

import (
	"context"
	"strings"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

func (h Handlers) CancelBooking(ctx context.Context, request CancelBookingRequestObject) (CancelBookingResponseObject, error) {
	userID := strings.TrimSpace(common.SafeDeref(request.Params.XUserId, ""))
	if userID == "" {
		return nil, common.NewUnauthorizedError("missing-user-id", "X-User-Id header is required")
	}

	if err := h.service.CancelBooking(ctx, app.CancelBooking{
		UserID:          userID,
		AppointmentUUID: request.AppointmentUuid,
	}); err != nil {
		return nil, err
	}

	return CancelBooking204Response{}, nil
}
