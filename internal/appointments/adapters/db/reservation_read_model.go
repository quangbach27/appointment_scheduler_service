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

	reservation, appointment := row.AppointmentsReservation, row.AppointmentsAppointment

	return httpport.GetReservationResponse{
		ReservationUuid: reservation.ReservationUuid,
		CreatedAt:       reservation.CreatedAt,
		ExpiredAt:       reservation.ExpiredAt,
		Appointment: httpport.AppointmentDetailResponse{
			AppointmentUuid:  appointment.AppointmentUuid,
			Status:           appointment.Status,
			DealerShipId:     appointment.DealerShipID,
			ServiceBayId:     appointment.ServiceBayID,
			TechnicianId:     appointment.TechnicianID,
			ServiceType:      appointment.ServiceType,
			VehicleUuid:      appointment.VehicleUuid,
			StartTime:        appointment.StartTime,
			EstimatedEndTime: appointment.EstimatedEndTime,
		},
	}, nil
}
