package component_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	appointmentClient "scheduler/internal/appointments/ports/http/client"
	"scheduler/internal/common"
)

func TestRequestBooking_Validation(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()

	t.Run("missing X-User-Id", func(t *testing.T) {
		t.Parallel()
		idempotencyKey := common.NewUUIDv7().String()
		resp, err := clients.Appointments.RequestBookingWithResponse(
			ctx,
			&appointmentClient.RequestBookingParams{
				IdempotencyKey: &idempotencyKey,
			},
			defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
		require.NotNil(t, resp.JSON401)
		assert.Equal(t, "missing-user-id", resp.JSON401.Slug)
	})

	t.Run("missing Idempotency-Key", func(t *testing.T) {
		t.Parallel()
		userID := common.NewUUIDv7().String()
		resp, err := clients.Appointments.RequestBookingWithResponse(
			ctx,
			&appointmentClient.RequestBookingParams{
				XUserId: &userID,
			},
			defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode())
		require.NotNil(t, resp.JSON400)
		assert.Equal(t, "missing-idempotency-key", resp.JSON400.Slug)
	})

	testCases := []struct {
		name       string
		mutate     func(*appointmentClient.RequestBookingJSONRequestBody)
		wantStatus int
		wantSlug   string
	}{
		{
			name:       "happy path",
			mutate:     nil,
			wantStatus: http.StatusCreated,
		},
		{
			name: "startTime zero",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.StartTime = time.Time{}
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-appointment",
		},
		{
			name: "startTime too soon",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.StartTime = time.Now().Add(30 * time.Minute)
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-appointment",
		},
		{
			name: "startTime too far out",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.StartTime = time.Now().Add(40000 * 24 * time.Hour)
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-appointment",
		},
		{
			name: "vehicle: empty modelName",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.Vehicle.ModelName = ""
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-vehicle",
		},
		{
			name: "vehicle: empty makeCode",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.Vehicle.MakeCode = ""
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-vehicle",
		},
		{
			name: "vehicle: modelYear too old",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.Vehicle.ModelYear = 1800
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-vehicle",
		},
		{
			name: "vehicle: modelYear too new",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.Vehicle.ModelYear = time.Now().Year() + 2
			},
			wantStatus: http.StatusBadRequest,
			wantSlug:   "invalid-vehicle",
		},
		{
			name: "unknown serviceType",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.ServiceType = "unknown-service"
			},
			wantStatus: http.StatusNotFound,
			wantSlug:   "service-type-not-found",
		},
		{
			name: "dealer-offday has no availability",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.DealerShipId = dealerOffDay
			},
			wantStatus: http.StatusConflict,
			wantSlug:   "no-availability",
		},
		{
			name: "dealer-maintenance has no availability",
			mutate: func(b *appointmentClient.RequestBookingJSONRequestBody) {
				b.DealerShipId = dealerMaint
			},
			wantStatus: http.StatusConflict,
			wantSlug:   "no-availability",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t))
			if tc.mutate != nil {
				tc.mutate(&body)
			}

			userID := common.NewUUIDv7().String()
			idempotencyKey := common.NewUUIDv7().String()
			resp, err := clients.Appointments.RequestBookingWithResponse(
				ctx,
				&appointmentClient.RequestBookingParams{
					XUserId:        &userID,
					IdempotencyKey: &idempotencyKey,
				},
				body,
			)
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, resp.StatusCode())

			switch tc.wantStatus {
			case http.StatusCreated:
				require.NotNil(t, resp.JSON201)
				assert.False(t, resp.JSON201.ReservationUuid.IsZero())
			case http.StatusBadRequest:
				require.NotNil(t, resp.JSON400)
				if tc.wantSlug == "invalid-vehicle" {
					require.NotNil(t, resp.JSON400.Details)
					require.NotEmpty(t, *resp.JSON400.Details)
					assert.Equal(t, tc.wantSlug, resp.JSON400.Slug)
				} else {
					assert.Equal(t, tc.wantSlug, resp.JSON400.Slug)
				}
			case http.StatusNotFound:
				require.NotNil(t, resp.JSON404)
				assert.Equal(t, tc.wantSlug, resp.JSON404.Slug)
			case http.StatusConflict:
				require.NotNil(t, resp.JSON409)
				assert.Equal(t, tc.wantSlug, resp.JSON409.Slug)
			}
		})
	}
}

func TestRequestBooking_IdempotencyKey(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()

	userID := common.NewUUIDv7().String()
	idempotencyKey := common.NewUUIDv7().String()
	body := defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t))

	first := requestBooking(ctx, t, clients, userID, idempotencyKey, body)

	t.Run("same key and identical body returns the same reservation", func(t *testing.T) {
		second := requestBooking(ctx, t, clients, userID, idempotencyKey, body)
		assert.Equal(t, first, second)
	})

	t.Run("same key with a different body still returns the original reservation", func(t *testing.T) {
		differentBody := defaultRequestBookingBody(dealerSolo, serviceOilChange, nextStart(t))
		replay := requestBooking(ctx, t, clients, userID, idempotencyKey, differentBody)
		assert.Equal(t, first, replay)
	})

	t.Run("a fresh key produces a different reservation", func(t *testing.T) {
		fresh := requestBooking(
			ctx, t, clients,
			common.NewUUIDv7().String(),
			common.NewUUIDv7().String(),
			defaultRequestBookingBody(dealerHappyPath, serviceOilChange, nextStart(t)),
		)
		assert.NotEqual(t, first, fresh)
	})
}

const concurrentBookers = 5

func TestRequestBooking_RaceCondition_OnlyOneBookerWins(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)
	ctx := context.Background()
	body := defaultRequestBookingBody(dealerSolo, serviceOilChange, nextStart(t))

	var wg sync.WaitGroup
	results := make([]*appointmentClient.RequestBookingClientResponse, concurrentBookers)

	wg.Add(concurrentBookers)
	for i := range concurrentBookers {
		go func(i int) {
			defer wg.Done()
			userID := common.NewUUIDv7().String()
			idempotencyKey := common.NewUUIDv7().String()
			resp, err := clients.Appointments.RequestBookingWithResponse(
				ctx,
				&appointmentClient.RequestBookingParams{
					XUserId:        &userID,
					IdempotencyKey: &idempotencyKey,
				},
				body,
			)
			require.NoError(t, err)
			results[i] = resp
		}(i)
	}
	wg.Wait()

	successes, failures := 0, 0
	for _, resp := range results {
		switch resp.StatusCode() {
		case http.StatusCreated:
			successes++
			require.NotNil(t, resp.JSON201)
		case http.StatusConflict:
			failures++
			require.NotNil(t, resp.JSON409)
			assert.Equal(t, "no-availability", resp.JSON409.Slug)
		default:
			t.Fatalf("unexpected status %d: %s", resp.StatusCode(), resp.Body)
		}
	}

	assert.Equal(t, 1, successes, "exactly one concurrent booking should win")
	assert.Equal(t, concurrentBookers-1, failures, "every other concurrent booking should lose with no-availability")
}
