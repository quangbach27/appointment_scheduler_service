//go:build integration

package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	appointmentsdb "scheduler/internal/appointments/adapters/db"
	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
	"scheduler/internal/common/testutils"
)

func newTestAppointmentAndReservation(t *testing.T, userID string) (*domain.Vehicle, *domain.Appointment, *domain.Reservation) {
	t.Helper()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: time.Hour,
		ReservationTTL:   5 * time.Minute,
	})

	vehicle, err := domain.NewVehicle("TOYOTA", "Camry", 2020, "ABC-123")
	require.NoError(t, err)

	appointment, err := factory.CreateAppointment(domain.AppointmentData{
		DealerShipID:      "dealer-1",
		ServiceBayID:      "bay-1",
		TechnicianID:      "tech-1",
		UserID:            userID,
		ServiceType:       "oil-change",
		VehicleUUID:       vehicle.UUID(),
		StartTime:         time.Now().Add(2 * time.Hour),
		EstimatedDuration: 30 * time.Minute,
	})
	require.NoError(t, err)

	reservation, err := factory.CreateReservation(appointment.UUID(), common.NewUUIDv7().String())
	require.NoError(t, err)

	return vehicle, appointment, reservation
}

func TestAppointmentRepository_RequestAppointment(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	userID := common.NewUUIDv7().String()
	vehicle, appointment, reservation := newTestAppointmentAndReservation(t, userID)

	reservationUUID, err := repo.RequestAppointment(context.Background(), userID, vehicle, appointment, reservation)
	require.NoError(t, err)
	assert.Equal(t, reservation.UUID(), reservationUUID)
}

func TestAppointmentRepository_RequestAppointment_DuplicateIdempotencyKeyReturnsOriginal(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	userID := common.NewUUIDv7().String()
	vehicle, appointment, reservation := newTestAppointmentAndReservation(t, userID)

	first, err := repo.RequestAppointment(context.Background(), userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	// A replay with a freshly minted appointment/reservation but the same idempotency key
	// must be rejected in favor of the original — no orphaned duplicate appointment.
	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{MinStartLeadTime: time.Hour})
	replayAppointment, err := factory.CreateAppointment(domain.AppointmentData{
		DealerShipID:      "dealer-1",
		ServiceBayID:      "bay-1",
		TechnicianID:      "tech-1",
		UserID:            userID,
		ServiceType:       "oil-change",
		VehicleUUID:       vehicle.UUID(),
		StartTime:         time.Now().Add(2 * time.Hour),
		EstimatedDuration: 30 * time.Minute,
	})
	require.NoError(t, err)
	replayReservation := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		replayAppointment.UUID(),
		time.Now(),
		time.Now().Add(5*time.Minute),
		reservation.IdempotencyKey(),
	)

	second, err := repo.RequestAppointment(context.Background(), userID, vehicle, replayAppointment, replayReservation)
	require.NoError(t, err)
	assert.Equal(t, first, second)

	// Confirming the returned reservation must resolve to the ORIGINAL appointment,
	// proving the replay's freshly minted appointment was never persisted.
	confirmedUUID, err := repo.ConfirmAppointment(context.Background(), userID, second)
	require.NoError(t, err)
	assert.Equal(t, appointment.UUID(), confirmedUUID)
	assert.NotEqual(t, replayAppointment.UUID(), confirmedUUID)
}

func TestAppointmentRepository_ConfirmAppointment(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	userID := common.NewUUIDv7().String()
	vehicle, appointment, reservation := newTestAppointmentAndReservation(t, userID)

	_, err := repo.RequestAppointment(context.Background(), userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	confirmedUUID, err := repo.ConfirmAppointment(context.Background(), userID, reservation.UUID())
	require.NoError(t, err)
	assert.Equal(t, appointment.UUID(), confirmedUUID)
}

func TestAppointmentRepository_ConfirmAppointment_ReservationNotFound(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())

	_, err := repo.ConfirmAppointment(context.Background(), common.NewUUIDv7().String(), domain.ReservationUUID{UUID: common.NewUUIDv7()})
	require.Error(t, err)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "reservation-not-found", domainErr.ErrorSlug)
}

func TestAppointmentRepository_ConfirmAppointment_WrongUserReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	userID := common.NewUUIDv7().String()
	vehicle, appointment, reservation := newTestAppointmentAndReservation(t, userID)

	_, err := repo.RequestAppointment(context.Background(), userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	_, err = repo.ConfirmAppointment(context.Background(), common.NewUUIDv7().String(), reservation.UUID())
	require.Error(t, err)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-not-found", domainErr.ErrorSlug)
}

func TestAppointmentRepository_CancelAppointment(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	userID := common.NewUUIDv7().String()
	vehicle, appointment, reservation := newTestAppointmentAndReservation(t, userID)

	_, err := repo.RequestAppointment(context.Background(), userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	_, err = repo.ConfirmAppointment(context.Background(), userID, reservation.UUID())
	require.NoError(t, err)

	err = repo.CancelAppointment(context.Background(), userID, appointment.UUID())
	require.NoError(t, err)
}

func TestAppointmentRepository_CancelAppointment_WhenNotConfirmed_ReturnsDomainError(t *testing.T) {
	t.Parallel()

	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	userID := common.NewUUIDv7().String()
	vehicle, appointment, reservation := newTestAppointmentAndReservation(t, userID)

	_, err := repo.RequestAppointment(context.Background(), userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	err = repo.CancelAppointment(context.Background(), userID, appointment.UUID())
	require.Error(t, err)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-not-confirmed", domainErr.ErrorSlug)
}
