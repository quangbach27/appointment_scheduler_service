package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler/internal/appointments/adapters/db/dbmodels"
	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
)

const uniqueReservationIdempotencyKeyConstraint = "uq_reservation_idempotency_key"

type AppointmentRepository struct {
	db *pgxpool.Pool
}

func NewAppointmentRepository(db *pgxpool.Pool) *AppointmentRepository {
	if db == nil {
		panic("pool can't be nil")
	}

	return &AppointmentRepository{db: db}
}

var _ domain.AppointmentRepository = (*AppointmentRepository)(nil)

func (r *AppointmentRepository) RequestAppointment(
	ctx context.Context,
	userID string,
	vehicle *domain.Vehicle,
	appointment *domain.Appointment,
	reservation *domain.Reservation,
) (domain.ReservationUUID, error) {
	var result domain.ReservationUUID

	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		q := dbmodels.New(tx)

		var licensePlate *string
		if lp := vehicle.LicensePlate(); lp != "" {
			licensePlate = &lp
		}

		if err := q.UpsertVehicle(ctx, dbmodels.UpsertVehicleParams{
			VehicleUuid:  vehicle.UUID(),
			UserID:       userID,
			MakeCode:     vehicle.MakeCode(),
			ModelName:    vehicle.ModelName(),
			ModelYear:    int16(vehicle.ModelYear()),
			LicensePlate: licensePlate,
		}); err != nil {
			return err
		}

		details := appointment.Details()
		if err := q.CreateAppointment(ctx, dbmodels.CreateAppointmentParams{
			AppointmentUuid:  appointment.UUID(),
			DealerShipID:     details.DealerShipID(),
			ServiceBayID:     details.ServiceBayID(),
			TechnicianID:     details.TechnicianID(),
			UserID:           details.UserID(),
			ServiceType:      details.ServiceType(),
			VehicleUuid:      details.VehicleUUID(),
			StartTime:        details.StartTime(),
			EstimatedEndTime: details.EstimatedEndTime(),
			Status:           appointment.Status(),
		}); err != nil {
			return err
		}

		created, err := q.CreateReservation(ctx, dbmodels.CreateReservationParams{
			ReservationUuid: reservation.UUID(),
			AppointmentUuid: reservation.AppointmentUUID(),
			CreatedAt:       reservation.CreatedAt(),
			ExpiredAt:       reservation.ExpiredAt(),
			IdempotencyKey:  reservation.IdempotencyKey(),
		})
		if err != nil {
			return err
		}

		result = created.ReservationUuid
		return nil
	})
	if err != nil {
		if common.IsUniqueViolationError(err, uniqueReservationIdempotencyKeyConstraint) {
			existing, getErr := dbmodels.New(r.db).GetReservationByIdempotencyKey(ctx, reservation.IdempotencyKey())
			if getErr != nil {
				return domain.ReservationUUID{}, getErr
			}
			return existing.ReservationUuid, nil
		}
		return domain.ReservationUUID{}, err
	}

	return result, nil
}

func (r *AppointmentRepository) ConfirmAppointment(
	ctx context.Context,
	userID string,
	reservationUUID domain.ReservationUUID,
) (domain.AppointmentUUID, error) {
	var appointmentUUID domain.AppointmentUUID

	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		q := dbmodels.New(tx)

		reservationRow, err := q.GetReservationByUUID(ctx, reservationUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return common.NewNotFoundError("reservation-not-found", "reservation not found")
			}
			return err
		}
		reservation := unmarshalReservation(reservationRow)

		appointmentRow, err := q.GetAppointmentByUUIDAndUserID(ctx, dbmodels.GetAppointmentByUUIDAndUserIDParams{
			AppointmentUuid: reservation.AppointmentUUID(),
			UserID:          userID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return common.NewNotFoundError("appointment-not-found", "appointment not found")
			}
			return err
		}
		appointment := unmarshalAppointment(appointmentRow)

		if err := appointment.Confirm(reservation); err != nil {
			return err
		}

		if err := q.UpdateAppointmentStatus(ctx, dbmodels.UpdateAppointmentStatusParams{
			AppointmentUuid: appointment.UUID(),
			Status:          appointment.Status(),
		}); err != nil {
			return err
		}

		appointmentUUID = appointment.UUID()
		return nil
	})
	if err != nil {
		return domain.AppointmentUUID{}, err
	}

	return appointmentUUID, nil
}

func (r *AppointmentRepository) CancelAppointment(
	ctx context.Context,
	userID string,
	uuid domain.AppointmentUUID,
) error {
	return common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		q := dbmodels.New(tx)

		appointmentRow, err := q.GetAppointmentByUUIDAndUserID(ctx, dbmodels.GetAppointmentByUUIDAndUserIDParams{
			AppointmentUuid: uuid,
			UserID:          userID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return common.NewNotFoundError("appointment-not-found", "appointment not found")
			}
			return err
		}
		appointment := unmarshalAppointment(appointmentRow)

		if err := appointment.Cancel(); err != nil {
			return err
		}

		return q.UpdateAppointmentStatus(ctx, dbmodels.UpdateAppointmentStatusParams{
			AppointmentUuid: appointment.UUID(),
			Status:          appointment.Status(),
		})
	})
}

func unmarshalReservation(row dbmodels.AppointmentsReservation) *domain.Reservation {
	return domain.UnmarshalReservation(
		row.ReservationUuid,
		row.AppointmentUuid,
		row.CreatedAt,
		row.ExpiredAt,
		row.IdempotencyKey,
	)
}

func unmarshalAppointment(row dbmodels.AppointmentsAppointment) *domain.Appointment {
	return domain.UnmarshalAppointment(
		row.AppointmentUuid,
		row.DealerShipID,
		row.ServiceBayID,
		row.TechnicianID,
		row.UserID,
		row.ServiceType,
		row.VehicleUuid,
		row.StartTime,
		row.EstimatedEndTime,
		row.Status,
	)
}
