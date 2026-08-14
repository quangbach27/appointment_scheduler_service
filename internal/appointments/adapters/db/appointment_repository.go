package db

import (
	"context"
	"errors"
	"fmt"

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

func (r *AppointmentRepository) FindReservationByIdempotencyKey(
	ctx context.Context,
	idempotencyKey string,
) (domain.ReservationUUID, bool, error) {
	existing, err := dbmodels.New(r.db).GetReservationByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ReservationUUID{}, false, nil
		}
		return domain.ReservationUUID{}, false, err
	}

	return existing.ReservationUuid, true, nil
}

func (r *AppointmentRepository) RequestAppointment(
	ctx context.Context,
	params domain.RequestAppointmentParams,
	createFn func(
		busyServiceBayIDs map[string]string,
		busyTechnicianIDs map[string]string,
	) (*domain.Appointment, *domain.Reservation, error),
) (domain.ReservationUUID, error) {
	var result domain.ReservationUUID

	err := common.UpdateInSerializableTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		q := dbmodels.New(tx)

		// Idempotency is resolved before availability: a replay must return its
		// original reservation, not be told its own booking made the slot busy.
		existing, err := q.GetReservationByIdempotencyKey(ctx, params.IdempotencyKey)
		if err == nil {
			result = existing.ReservationUuid
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		busyServiceBayIDs, busyTechnicianIDs, err := r.getBusyCandidates(ctx, q, params)
		if err != nil {
			return err
		}

		appointment, reservation, err := createFn(busyServiceBayIDs, busyTechnicianIDs)
		if err != nil {
			return err
		}
		details := appointment.Details()

		var licensePlate *string
		if lp := params.Vehicle.LicensePlate(); lp != "" {
			licensePlate = &lp
		}

		if err := q.UpsertVehicle(ctx, dbmodels.UpsertVehicleParams{
			VehicleUuid:  params.Vehicle.UUID(),
			UserID:       params.UserID,
			MakeCode:     params.Vehicle.MakeCode(),
			ModelName:    params.Vehicle.ModelName(),
			ModelYear:    int16(params.Vehicle.ModelYear()),
			LicensePlate: licensePlate,
		}); err != nil {
			return err
		}

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

		if err := q.CreateReservation(ctx, dbmodels.CreateReservationParams{
			ReservationUuid: reservation.UUID(),
			AppointmentUuid: reservation.AppointmentUUID(),
			CreatedAt:       reservation.CreatedAt(),
			ExpiredAt:       reservation.ExpiredAt(),
			IdempotencyKey:  reservation.IdempotencyKey(),
		}); err != nil {
			return err
		}

		result = reservation.UUID()
		return nil
	})
	if err != nil {
		// Backstop for two identical requests racing past the in-tx lookup above.
		if common.IsUniqueViolationError(err, uniqueReservationIdempotencyKeyConstraint) {
			existing, getErr := dbmodels.New(r.db).GetReservationByIdempotencyKey(ctx, params.IdempotencyKey)
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

		reservation, err := r.getReservation(ctx, q, reservationUUID)
		if err != nil {
			return err
		}

		appointment, err := r.getAppointment(ctx, q, reservation.AppointmentUUID(), userID)
		if err != nil {
			return err
		}

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
	appointmentUUID domain.AppointmentUUID,
) error {
	return common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		q := dbmodels.New(tx)

		appointment, err := r.getAppointment(ctx, q, appointmentUUID, userID)
		if err != nil {
			return err
		}

		if err := appointment.Cancel(); err != nil {
			return err
		}

		return q.UpdateAppointmentStatus(ctx, dbmodels.UpdateAppointmentStatusParams{
			AppointmentUuid: appointment.UUID(),
			Status:          appointment.Status(),
		})
	})
}

func (r *AppointmentRepository) getBusyCandidates(
	ctx context.Context,
	queries *dbmodels.Queries,
	params domain.RequestAppointmentParams,
) (
	map[string]string,
	map[string]string,
	error,
) {
	busyRows, err := queries.GetBusyServiceBaysAndTechnicians(ctx, dbmodels.GetBusyServiceBaysAndTechniciansParams{
		DealerShipID: params.DealerShipID,
		StartTime:    params.StartTime,
		EndTime:      params.EndTime,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving busy candidate: %w", err)
	}

	busyServiceBayIDs := make(map[string]string, len(busyRows))
	busyTechnicianIDs := make(map[string]string, len(busyRows))
	for _, row := range busyRows {
		busyServiceBayIDs[row.ServiceBayID] = row.TechnicianID
		busyTechnicianIDs[row.TechnicianID] = row.ServiceBayID
	}

	return busyServiceBayIDs, busyTechnicianIDs, nil
}

func (r *AppointmentRepository) getReservation(
	ctx context.Context,
	q *dbmodels.Queries,
	reservationUUID domain.ReservationUUID,
) (*domain.Reservation, error) {
	reservationRow, err := q.GetReservationByUUID(ctx, reservationUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, common.NewNotFoundError(
				"reservation-not-found",
				"reservation not found",
			).WithInternalError(err)
		}
		return nil, fmt.Errorf("error retrieving reservation: %w", err)
	}

	return unmarshalReservation(reservationRow), nil
}

func (r *AppointmentRepository) getAppointment(
	ctx context.Context,
	q *dbmodels.Queries,
	appointmentUUID domain.AppointmentUUID,
	userID string,
) (*domain.Appointment, error) {
	appointmentRow, err := q.GetAppointmentByUUIDAndUserID(ctx, dbmodels.GetAppointmentByUUIDAndUserIDParams{
		AppointmentUuid: appointmentUUID,
		UserID:          userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, common.NewNotFoundError(
				"appointment-not-found",
				"appointment not found",
			).WithInternalError(err)
		}
		return nil, fmt.Errorf("error retrieving appointment: %w", err)
	}

	return unmarshalAppointment(appointmentRow), nil
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
