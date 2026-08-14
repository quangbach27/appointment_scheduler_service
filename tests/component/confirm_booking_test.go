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

// reservation-mismatch (400) and reservation-expired (410) are intentionally
// not covered here: reservation-mismatch isn't reachable through the public
// HTTP surface, and reservation-expired would require waiting out the real
// 5-minute TTL (.env.test, no clock injection exists). Both are covered at
// the domain-unit level in internal/appointments/domain/appointment_test.go.
func TestConfirmBooking_Validation(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()

	t.Run("missing X-User-Id", func(t *testing.T) {
		t.Parallel()
		resp, err := clients.Appointments.ConfirmBookingWithResponse(
			ctx,
			appointmentClient.ReservationUUID{UUID: common.NewUUIDv7()},
			&appointmentClient.ConfirmBookingParams{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
		require.NotNil(t, resp.JSON401)
		assert.Equal(t, "not-authenticated", resp.JSON401.Slug)
	})

	t.Run("reservation not found", func(t *testing.T) {
		t.Parallel()
		userID := common.NewUUIDv7().String()
		resp, err := clients.Appointments.ConfirmBookingWithResponse(
			ctx,
			appointmentClient.ReservationUUID{UUID: common.NewUUIDv7()},
			&appointmentClient.ConfirmBookingParams{XUserId: &userID},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode())
		require.NotNil(t, resp.JSON404)
		assert.Equal(t, "reservation-not-found", resp.JSON404.Slug)
	})
}

func TestConfirmBooking_HappyPath(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)

	appointmentUUID := confirmBooking(ctx, t, clients, reservationUUID, userID)
	assert.False(t, appointmentUUID.IsZero())
}

func TestConfirmBooking_WrongUser(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)

	wrongUserID := common.NewUUIDv7().String()
	resp, err := clients.Appointments.ConfirmBookingWithResponse(
		ctx,
		reservationUUID,
		&appointmentClient.ConfirmBookingParams{XUserId: &wrongUserID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode())
	require.NotNil(t, resp.JSON404)
	assert.Equal(t, "appointment-not-found", resp.JSON404.Slug)
}

func TestConfirmBooking_DoubleConfirm(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()
	userID := common.NewUUIDv7().String()

	reservationUUID := requestBooking(
		ctx, t, clients, userID, common.NewUUIDv7().String(),
		defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
	)
	confirmBooking(ctx, t, clients, reservationUUID, userID)

	second, err := clients.Appointments.ConfirmBookingWithResponse(
		ctx,
		reservationUUID,
		&appointmentClient.ConfirmBookingParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, second.StatusCode())
	require.NotNil(t, second.JSON409)
	assert.Equal(t, "appointment-not-pending", second.JSON409.Slug)
}
