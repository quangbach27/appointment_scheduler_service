package http

import (
	"context"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"scheduler/internal/appointments/app"
	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
)

func (h Handlers) ConfirmAppointment(ctx context.Context, request ConfirmAppointmentRequestObject) (ConfirmAppointmentResponseObject, error) {
	userID := strings.TrimSpace(request.Params.XUserId)
	if userID == "" {
		return nil, common.NewUnauthorizedError("missing-user-id", "X-User-Id header is required")
	}

	appointmentUUID, err := h.service.ConfirmAppointment(ctx, app.ConfirmAppointment{
		UserID: userID,
		ReservationUUID: domain.ReservationUUID{
			UUID: common.UUID(request.ReservationUuid),
		},
	})
	if err != nil {
		return nil, err
	}

	return ConfirmAppointment200JSONResponse{
		AppointmentUuid: openapi_types.UUID(appointmentUUID.UUID),
		Status:          Confirmed,
	}, nil
}
