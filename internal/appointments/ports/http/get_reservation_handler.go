package http

import (
	"context"
	"strings"

	"scheduler/internal/common"
)

func (h Handlers) GetReservation(ctx context.Context, request GetReservationRequestObject) (GetReservationResponseObject, error) {
	userID := strings.TrimSpace(common.SafeDeref(request.Params.XUserId, ""))
	if userID == "" {
		return nil, common.NewUnauthorizedError("missing-user-id", "X-User-Id header is required")
	}

	response, err := h.reservationReadModel.GetReservationByID(ctx, userID, request.ReservationUuid)
	if err != nil {
		return nil, err
	}

	return GetReservation200JSONResponse(response), nil
}
