package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler/internal/appointments/adapters/db/dbmodels"
	"scheduler/internal/appointments/domain"
	httpport "scheduler/internal/appointments/ports/http"
	"scheduler/internal/common"
)

type ReservationReadModel struct {
	db *pgxpool.Pool
}

func NewReservationReadModel(db *pgxpool.Pool) *ReservationReadModel {
	if db == nil {
		panic("pool can't be nil")
	}

	return &ReservationReadModel{db: db}
}

var _ httpport.ReservationReadModel = (*ReservationReadModel)(nil)

func (r *ReservationReadModel) GetReservationByID(
	ctx context.Context,
	userID string,
	reservationUUID domain.ReservationUUID,
) (httpport.GetReservationResponse, error) {
	row, err := dbmodels.New(r.db).GetReservationWithAppointmentByUUID(ctx, dbmodels.GetReservationWithAppointmentByUUIDParams{
		ReservationUuid: reservationUUID,
		UserID:          userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpport.GetReservationResponse{}, common.NewNotFoundError("reservation-not-found", "reservation not found")
		}
		return httpport.GetReservationResponse{}, err
	}

	return httpport.GetReservationResponse{
		ReservationUuid: row.ReservationUuid,
		ExpiredAt:       row.ExpiredAt,
		Appointment: httpport.AppointmentDetailResponse{
			AppointmentUuid:  row.AppointmentUuid,
			Status:           row.Status,
			DealerShipId:     row.DealerShipID,
			ServiceBayId:     row.ServiceBayID,
			TechnicianId:     row.TechnicianID,
			ServiceType:      row.ServiceType,
			VehicleUuid:      row.VehicleUuid,
			StartTime:        row.StartTime,
			EstimatedEndTime: row.EstimatedEndTime,
		},
	}, nil
}
