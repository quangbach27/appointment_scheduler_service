package domain_test

import (
	"testing"
	"time"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"

	"github.com/go-openapi/testify/v2/assert"
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
