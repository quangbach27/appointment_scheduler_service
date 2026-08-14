package http

import (
	"context"
	"strings"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

func (h Handlers) RequestBooking(ctx context.Context, request RequestBookingRequestObject) (RequestBookingResponseObject, error) {
	userID, err := requireUserID(request.Params.XUserId)
	if err != nil {
		return nil, err
	}

	idempotencyKey := strings.TrimSpace(common.SafeDeref(request.Params.IdempotencyKey, ""))
	if idempotencyKey == "" {
		return nil, common.NewInvalidInputError("missing-idempotency-key", "Idempotency-Key header is required")
	}

	if request.Body == nil {
		return nil, common.NewInvalidInputError("missing-body", "request body is required")
	}

	reservationUUID, err := h.service.RequestBooking(ctx, app.RequestBooking{
		UserID:         userID,
		DealerShipID:   request.Body.DealerShipId,
		ServiceType:    request.Body.ServiceType,
		StartTime:      request.Body.StartTime,
		IdempotencyKey: idempotencyKey,
		Vehicle: app.RequestBookingVehicle{
			MakeCode:     request.Body.Vehicle.MakeCode,
			ModelName:    request.Body.Vehicle.ModelName,
			ModelYear:    request.Body.Vehicle.ModelYear,
			LicensePlate: common.SafeDeref(request.Body.Vehicle.LicensePlate, ""),
		},
	})
	if err != nil {
		return nil, err
	}

	return RequestBooking201JSONResponse{ReservationUuid: reservationUUID}, nil
}
