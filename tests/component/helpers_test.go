package component_test

import (
	"context"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"

	appointmentClient "scheduler/internal/appointments/ports/http/client"
)

const (
	dealerHappyPath = "dealer-1"           // multiple techs/bays; tech-2 (score 4.8) should win when free
	dealerSolo      = "dealer-solo"        // exactly one tech + one bay — used to exhaust availability
	dealerOffDay    = "dealer-offday"      // technician permanently blacked out
	dealerMaint     = "dealer-maintenance" // bay permanently blacked out

	serviceOilChange = "oil-change" // 1h, no required certs — safe default
)

var startCounter atomic.Int64

// nextStart returns a start time far enough in the future (per
// APPOINTMENT_MAX_START_LEAD_TIME_DAYS=36500 in .env.test) that two tests
// booking the same dealer never contend for the same technician/bay window
// in the shared, long-lived component test DB. The day offset is randomized
// rather than derived from wall-clock time or a plain call counter: separate
// test-binary invocations run close together in time would otherwise land
// on nearly identical windows and collide with rows a previous run left
// behind. The counter still guarantees distinctness between calls within a
// single run (e.g. two windows requested a moment apart in the same test).
func nextStart(t *testing.T) time.Time {
	t.Helper()
	base := 400 * 24 * time.Hour // safely clear of the 1h minimum lead time
	jitter := time.Duration(rand.Int64N(35000)) * 24 * time.Hour
	call := time.Duration(startCounter.Add(1)) * time.Minute
	return time.Now().Add(base + jitter + call).Truncate(time.Second)
}

func defaultVehicle() appointmentClient.RequestBookingVehicle {
	return appointmentClient.RequestBookingVehicle{
		MakeCode:  "TOYT",
		ModelName: "Camry",
		ModelYear: 2022,
	}
}

func defaultRequestBookingBody(dealerShipID, serviceType string, startTime time.Time) appointmentClient.RequestBookingJSONRequestBody {
	return appointmentClient.RequestBookingJSONRequestBody{
		DealerShipId: dealerShipID,
		ServiceType:  serviceType,
		StartTime:    startTime,
		Vehicle:      defaultVehicle(),
	}
}

// requestBooking sends one RequestBooking call, asserts it succeeded, and
// returns the resulting reservation UUID. Call sites that expect a
// non-success outcome must call
// clients.Appointments.RequestBookingWithResponse directly instead (see the
// header-validation subtests in request_booking_test.go).
func requestBooking(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	userID string,
	idempotencyKey string,
	body appointmentClient.RequestBookingJSONRequestBody,
) appointmentClient.ReservationUUID {
	t.Helper()

	resp, err := clients.Appointments.RequestBookingWithResponse(
		ctx,
		&appointmentClient.RequestBookingParams{
			XUserId:        &userID,
			IdempotencyKey: &idempotencyKey,
		},
		body,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode(), "unexpected response: %s", resp.Body)
	require.NotNil(t, resp.JSON201)
	return resp.JSON201.ReservationUuid
}

// confirmAppointment sends one ConfirmAppointment call, asserts it
// succeeded, and returns the resulting appointment UUID. Call sites that
// expect a non-success outcome must call
// clients.Appointments.ConfirmAppointmentWithResponse directly instead.
func confirmAppointment(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	reservationUUID appointmentClient.ReservationUUID,
	userID string,
) appointmentClient.AppointmentUUID {
	t.Helper()

	resp, err := clients.Appointments.ConfirmAppointmentWithResponse(
		ctx,
		reservationUUID,
		&appointmentClient.ConfirmAppointmentParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "unexpected response: %s", resp.Body)
	require.NotNil(t, resp.JSON200)
	return resp.JSON200.AppointmentUuid
}

// getReservation sends one GetReservation call, asserts it succeeded, and
// returns the reservation+appointment detail payload.
func getReservation(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	reservationUUID appointmentClient.ReservationUUID,
	userID string,
) appointmentClient.GetReservationResponse {
	t.Helper()

	resp, err := clients.Appointments.GetReservationWithResponse(
		ctx,
		reservationUUID,
		&appointmentClient.GetReservationParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "unexpected response: %s", resp.Body)
	require.NotNil(t, resp.JSON200)
	return *resp.JSON200
}

// cancelAppointment sends one CancelAppointment call and asserts it
// succeeded (204, empty body). Used at setup/happy-path call sites where a
// non-success outcome is a test bug, not something under test.
func cancelAppointment(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	appointmentUUID appointmentClient.AppointmentUUID,
	userID string,
) {
	t.Helper()

	resp, err := clients.Appointments.CancelAppointmentWithResponse(
		ctx,
		appointmentUUID,
		&appointmentClient.CancelAppointmentParams{XUserId: &userID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode(), "unexpected response: %s", resp.Body)
}
