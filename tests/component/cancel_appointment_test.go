package component_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	appointmentClient "scheduler/internal/appointments/ports/http/client"
	"scheduler/internal/common"
)

// appointment-in-the-past (400) is intentionally not covered here: it
// compares the appointment's startTime against the real clock, and no clock
// injection exists anywhere in the codebase, so it isn't reachable in a
// fast component test. It's covered at the domain-unit level in
// internal/appointments/domain/appointment_test.go
// (TestAppointment_Cancel_WhenInPast_ReturnsError).
func TestCancelAppointment_Validation(t *testing.T) {
	clients := newTestClients(t)
	ctx := context.Background()

	t.Run("missing X-User-Id", func(t *testing.T) {
		resp, err := clients.Appointments.CancelAppointmentWithResponse(
			ctx,
			appointmentClient.AppointmentUUID{UUID: common.NewUUIDv7()},
			&appointmentClient.CancelAppointmentParams{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
		require.NotNil(t, resp.JSON401)
		assert.Equal(t, "missing-user-id", resp.JSON401.Slug)
	})

	t.Run("appointment not found", func(t *testing.T) {
		userID := common.NewUUIDv7().String()
		resp, err := clients.Appointments.CancelAppointmentWithResponse(
			ctx,
			appointmentClient.AppointmentUUID{UUID: common.NewUUIDv7()},
			&appointmentClient.CancelAppointmentParams{XUserId: &userID},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode())
		require.NotNil(t, resp.JSON404)
		assert.Equal(t, "appointment-not-found", resp.JSON404.Slug)
	})
}

func TestCancelAppointment_HappyPath(t *testing.T) {
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)
	appointmentUUID := confirmAppointment(ctx, t, clients, reservationUUID, userID)

	resp, err := clients.Appointments.CancelAppointmentWithResponse(
		ctx, appointmentUUID,
		&appointmentClient.CancelAppointmentParams{XUserId: &userID},
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode())
	assert.Empty(t, resp.Body)
}

func TestCancelAppointment_WrongUser(t *testing.T) {
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)
	appointmentUUID := confirmAppointment(ctx, t, clients, reservationUUID, userID)

	wrongUserID := common.NewUUIDv7().String()
	resp, err := clients.Appointments.CancelAppointmentWithResponse(
		ctx, appointmentUUID,
		&appointmentClient.CancelAppointmentParams{XUserId: &wrongUserID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode())
	require.NotNil(t, resp.JSON404)
	assert.Equal(t, "appointment-not-found", resp.JSON404.Slug)
}

func TestCancelAppointment_NotConfirmed(t *testing.T) {
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	// Book but never confirm — the appointment row exists (pending) from
	// the moment of booking; fetch its UUID via GetReservation since
	// RequestBooking's own response only exposes the reservation UUID.
	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)

	reservation := getReservation(ctx, t, clients, reservationUUID, userID)
	appointmentUUID := reservation.Appointment.AppointmentUuid

	resp, err := clients.Appointments.CancelAppointmentWithResponse(
		ctx, appointmentUUID,
		&appointmentClient.CancelAppointmentParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode())
	require.NotNil(t, resp.JSON409)
	assert.Equal(t, "appointment-not-confirmed", resp.JSON409.Slug)
}

func TestCancelAppointment_DoubleCancel(t *testing.T) {
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)
	appointmentUUID := confirmAppointment(ctx, t, clients, reservationUUID, userID)

	cancelAppointment(ctx, t, clients, appointmentUUID, userID)

	second, err := clients.Appointments.CancelAppointmentWithResponse(
		ctx, appointmentUUID,
		&appointmentClient.CancelAppointmentParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, second.StatusCode())
	require.NotNil(t, second.JSON409)
	assert.Equal(t, "appointment-already-cancelled", second.JSON409.Slug)
}
