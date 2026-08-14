package component_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	appointmentClient "scheduler/internal/appointments/ports/http/client"
	"scheduler/internal/common"
)

func TestCriticalFlow_BookRetrieveConfirm(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)

	retrieved := getReservation(ctx, t, clients, reservationUUID, userID)
	assert.Equal(t, "pending", retrieved.Appointment.Status.String())
	assert.True(t, retrieved.ExpiredAt.After(time.Now()))

	appointmentUUID := confirmAppointment(ctx, t, clients, reservationUUID, userID)
	assert.False(t, appointmentUUID.IsZero())

	confirmed := getReservation(ctx, t, clients, reservationUUID, userID)
	assert.Equal(t, "confirmed", confirmed.Appointment.Status.String())
	assert.Equal(t, appointmentUUID, confirmed.Appointment.AppointmentUuid)
}

func TestCriticalFlow_ConfirmFailsWhenReservationExpired(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)

	retrieved := getReservation(ctx, t, clients, reservationUUID, userID)

	// Wait past the reservation's actual expiry (APPOINTMENT_RESERVATION_TTL_SECONDS=30
	// in .env.test) rather than a hardcoded duration, so this stays correct
	// if the TTL is retuned later.
	time.Sleep(time.Until(retrieved.ExpiredAt) + 2*time.Second)

	resp, err := clients.Appointments.ConfirmAppointmentWithResponse(
		ctx, reservationUUID,
		&appointmentClient.ConfirmAppointmentParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusGone, resp.StatusCode())
	require.NotNil(t, resp.JSON410)
	assert.Equal(t, "reservation-expired", resp.JSON410.Slug)
}
