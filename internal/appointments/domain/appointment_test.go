package domain_test

import (
	"testing"
	"time"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestAppointment_Cancel(t *testing.T) {
	t.Parallel()

	appointment := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusConfirmed,
	)

	require.NoError(t, appointment.Cancel())
	assert.True(t, appointment.IsCancelled())
	assert.Equal(t, domain.AppointmentStatusCanceled, appointment.Status())
}

func TestAppointment_Cancel_WhenInPast_ReturnsError(t *testing.T) {
	t.Parallel()

	appointment := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
		domain.AppointmentStatusConfirmed,
	)

	err := appointment.Cancel()
	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-in-the-past", domainErr.ErrorSlug)
	assert.False(t, appointment.IsCancelled())
}

func TestAppointment_Confirm(t *testing.T) {
	t.Parallel()

	appointmentUUID := domain.AppointmentUUID{UUID: common.NewUUIDv7()}
	appointment := domain.UnmarshalAppointment(
		appointmentUUID,
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusPending,
	)
	reservation := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		appointmentUUID,
		time.Now(),
		time.Now().Add(5*time.Minute),
		"idem-key-1",
	)

	require.NoError(t, appointment.Confirm(reservation))
	assert.Equal(t, domain.AppointmentStatusConfirmed, appointment.Status())
}

func TestAppointment_Confirm_WhenReservationExpired_ReturnsError(t *testing.T) {
	t.Parallel()

	appointmentUUID := domain.AppointmentUUID{UUID: common.NewUUIDv7()}
	appointment := domain.UnmarshalAppointment(
		appointmentUUID,
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusPending,
	)
	reservation := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		appointmentUUID,
		time.Now().Add(-10*time.Minute),
		time.Now().Add(-1*time.Minute),
		"idem-key-2",
	)

	err := appointment.Confirm(reservation)
	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "reservation-expired", domainErr.ErrorSlug)
	assert.Equal(t, domain.AppointmentStatusPending, appointment.Status())
}

func TestAppointment_Confirm_WhenReservationMismatched_ReturnsError(t *testing.T) {
	t.Parallel()

	appointment := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusPending,
	)
	reservation := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		time.Now(),
		time.Now().Add(5*time.Minute),
		"idem-key-3",
	)

	err := appointment.Confirm(reservation)
	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "reservation-mismatch", domainErr.ErrorSlug)
}

func TestAppointment_Confirm_WhenNotPending_ReturnsError(t *testing.T) {
	t.Parallel()

	appointmentUUID := domain.AppointmentUUID{UUID: common.NewUUIDv7()}
	appointment := domain.UnmarshalAppointment(
		appointmentUUID,
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusConfirmed,
	)
	reservation := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		appointmentUUID,
		time.Now(),
		time.Now().Add(5*time.Minute),
		"idem-key-4",
	)

	err := appointment.Confirm(reservation)
	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-not-pending", domainErr.ErrorSlug)
}

func TestAppointment_IsExpired(t *testing.T) {
	t.Parallel()

	pendingInPast := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
		domain.AppointmentStatusPending,
	)
	assert.True(t, pendingInPast.IsExpired())

	pendingInFuture := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusPending,
	)
	assert.False(t, pendingInFuture.IsExpired())

	confirmedInPast := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
		domain.AppointmentStatusConfirmed,
	)
	assert.False(t, confirmedInPast.IsExpired())
}

func TestAppointment_Cancel_WhenNotConfirmed_ReturnsError(t *testing.T) {
	t.Parallel()

	appointment := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusPending,
	)

	err := appointment.Cancel()
	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-not-confirmed", domainErr.ErrorSlug)
	assert.False(t, appointment.IsCancelled())
}

func TestAppointment_Cancel_WhenAlreadyCancelled_ReturnsError(t *testing.T) {
	t.Parallel()

	appointment := domain.UnmarshalAppointment(
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		"dealer-123",
		"bay-5",
		"tech-9",
		"user-42",
		"oil-change",
		domain.VehicleUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(2*time.Hour),
		time.Now().Add(3*time.Hour),
		domain.AppointmentStatusCanceled,
	)

	err := appointment.Cancel()
	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-already-cancelled", domainErr.ErrorSlug)
}
