package domain

import (
	"context"
	"time"

	"scheduler/internal/common"
)

type RequestAppointmentParams struct {
	UserID         string
	IdempotencyKey string // this help to prevent over request booking due to networking issue
	DealerShipID   string
	StartTime      time.Time
	EndTime        time.Time
	Vehicle        *Vehicle
}

type AppointmentRepository interface {
	FindReservationByIdempotencyKey(
		ctx context.Context,
		idempotencyKey string,
	) (reservationUUID ReservationUUID, found bool, err error)

	RequestAppointment(
		ctx context.Context,
		params RequestAppointmentParams,
		createFn func(
			busyServiceBayIDs map[string]string,
			busyTechnicianIDs map[string]string,
		) (*Appointment, *Reservation, error),
	) (ReservationUUID, error)

	ConfirmAppointment(
		ctx context.Context,
		userID string,
		reservationUUID ReservationUUID,
	) (AppointmentUUID, error)

	CancelAppointment(
		ctx context.Context,
		userID string,
		uuid AppointmentUUID,
	) error
}

type AppointmentUUID struct {
	common.UUID
}

type Appointment struct {
	uuid    AppointmentUUID
	status  AppointmentStatus
	details AppointmentDetails
}

func (a *Appointment) UUID() AppointmentUUID {
	return a.uuid
}

func (a *Appointment) Status() AppointmentStatus {
	return a.status
}

func (a *Appointment) Details() AppointmentDetails {
	return a.details
}

func (a *Appointment) IsCancelled() bool {
	return a.status == AppointmentStatusCanceled
}

func (a *Appointment) IsExpired() bool {
	return a.status == AppointmentStatusPending && a.details.startTime.Before(time.Now())
}

func (a *Appointment) Confirm(reservation *Reservation) error {
	if reservation == nil || reservation.AppointmentUUID() != a.uuid {
		return common.NewInvalidInputError("reservation-mismatch", "reservation does not belong to this appointment")
	}

	if a.status != AppointmentStatusPending {
		return common.NewConflictError("appointment-not-pending", "appointment is not pending confirmation")
	}

	if reservation.IsExpired() {
		return common.NewExpiredError("reservation-expired", "reservation expired at %s", reservation.ExpiredAt().Format(time.RFC3339))
	}

	a.status = AppointmentStatusConfirmed
	return nil
}

func (a *Appointment) Cancel() error {
	if a.IsCancelled() {
		return common.NewConflictError("appointment-already-cancelled", "appointment is already cancelled")
	}

	if a.status != AppointmentStatusConfirmed {
		return common.NewConflictError("appointment-not-confirmed", "only confirmed appointments can be cancelled")
	}

	if a.details.startTime.Before(time.Now()) {
		return common.NewInvalidInputError("appointment-in-the-past", "appointment is in the past")
	}

	a.status = AppointmentStatusCanceled
	return nil
}
