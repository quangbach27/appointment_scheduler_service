package domain_test

import (
	"testing"
	"time"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestReservation_IsExpired(t *testing.T) {
	t.Parallel()

	notExpired := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(-1*time.Minute),
		time.Now().Add(1*time.Minute),
		"idem-key-1",
	)
	assert.False(t, notExpired.IsExpired())

	expired := domain.UnmarshalReservation(
		domain.ReservationUUID{UUID: common.NewUUIDv7()},
		domain.AppointmentUUID{UUID: common.NewUUIDv7()},
		time.Now().Add(-10*time.Minute),
		time.Now().Add(-1*time.Minute),
		"idem-key-2",
	)
	assert.True(t, expired.IsExpired())
}

func TestReservationFactory_CreateReservation(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		ReservationTTL: 10 * time.Minute,
	})

	appointmentUUID := domain.AppointmentUUID{UUID: common.NewUUIDv7()}
	reservation, err := factory.CreateReservation(appointmentUUID, "idem-key-3")

	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.Equal(t, appointmentUUID, reservation.AppointmentUUID())
	assert.Equal(t, "idem-key-3", reservation.IdempotencyKey())
	assert.False(t, reservation.IsExpired())
	assert.WithinDuration(t, time.Now().Add(10*time.Minute), reservation.ExpiredAt(), 2*time.Second)
}

func TestReservationFactory_CreateReservation_DefaultsAreApplied(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{})

	reservation, err := factory.CreateReservation(domain.AppointmentUUID{UUID: common.NewUUIDv7()}, "idem-key-4")

	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), reservation.ExpiredAt(), 2*time.Second)
}

func TestReservationFactory_CreateReservation_RequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{})

	reservation, err := factory.CreateReservation(domain.AppointmentUUID{UUID: common.NewUUIDv7()}, "  ")
	require.Error(t, err)
	assert.Nil(t, reservation)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "invalid-reservation", domainErr.ErrorSlug)
}
