package http

import (
	"context"
)

func (h Handlers) GetReservation(ctx context.Context, request GetReservationRequestObject) (GetReservationResponseObject, error) {
	userID, err := requireUserID(request.Params.XUserId)
	if err != nil {
		return nil, err
	}

	response, err := h.reservationReadModel.GetReservationByID(ctx, userID, request.ReservationUuid)
	if err != nil {
		return nil, err
	}

	return GetReservation200JSONResponse(response), nil
}
