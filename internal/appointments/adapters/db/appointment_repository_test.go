//go:build integration

package db_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	appointmentsdb "scheduler/internal/appointments/adapters/db"
	"scheduler/internal/appointments/adapters/db/dbmodels"
	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
	"scheduler/internal/common/testutils"
)

func TestAppointmentRepository_ConfirmAppointment(t *testing.T) {
	t.Parallel()
	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	f := newAppointmentFixture(t)
	vehicle, appointment, reservation := f.build(t)

	_, err := requestAppointment(t, repo, f.userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	confirmedUUID, err := repo.ConfirmAppointment(context.Background(), f.userID, reservation.UUID())
	require.NoError(t, err)
	assert.Equal(t, appointment.UUID(), confirmedUUID)
}

func TestAppointmentRepository_ConfirmAppointment_ReservationNotFound(t *testing.T) {
	t.Parallel()
	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())

	_, err := repo.ConfirmAppointment(context.Background(), testutils.RandomString(), domain.ReservationUUID{UUID: common.NewUUIDv7()})
	require.Error(t, err)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "reservation-not-found", domainErr.ErrorSlug)
}

func TestAppointmentRepository_ConfirmAppointment_WrongUserReturnsNotFound(t *testing.T) {
	t.Parallel()
	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	f := newAppointmentFixture(t)
	vehicle, appointment, reservation := f.build(t)

	_, err := requestAppointment(t, repo, f.userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	_, err = repo.ConfirmAppointment(context.Background(), testutils.RandomString(), reservation.UUID())
	require.Error(t, err)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-not-found", domainErr.ErrorSlug)
}

func TestAppointmentRepository_CancelAppointment(t *testing.T) {
	t.Parallel()
	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	f := newAppointmentFixture(t)
	vehicle, appointment, reservation := f.build(t)

	_, err := requestAppointment(t, repo, f.userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	_, err = repo.ConfirmAppointment(context.Background(), f.userID, reservation.UUID())
	require.NoError(t, err)

	err = repo.CancelAppointment(context.Background(), f.userID, appointment.UUID())
	require.NoError(t, err)
}

func TestAppointmentRepository_CancelAppointment_WhenNotConfirmed_ReturnsDomainError(t *testing.T) {
	t.Parallel()
	repo := appointmentsdb.NewAppointmentRepository(testutils.NewDB())
	f := newAppointmentFixture(t)
	vehicle, appointment, reservation := f.build(t)

	_, err := requestAppointment(t, repo, f.userID, vehicle, appointment, reservation)
	require.NoError(t, err)

	err = repo.CancelAppointment(context.Background(), f.userID, appointment.UUID())
	require.Error(t, err)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "appointment-not-confirmed", domainErr.ErrorSlug)
}

// requestAppointment drives the repository with a createFn that just returns
// the pre-built appointment/reservation, ignoring the busy maps: these tests
// exercise confirm/cancel, not availability conflicts.
func requestAppointment(
	t *testing.T,
	repo *appointmentsdb.AppointmentRepository,
	userID string,
	vehicle *domain.Vehicle,
	appointment *domain.Appointment,
	reservation *domain.Reservation,
) (domain.ReservationUUID, error) {
	t.Helper()

	details := appointment.Details()
	return repo.RequestAppointment(
		context.Background(),
		domain.RequestAppointmentParams{
			UserID:         userID,
			IdempotencyKey: reservation.IdempotencyKey(),
			DealerShipID:   details.DealerShipID(),
			StartTime:      details.StartTime(),
			EndTime:        details.EstimatedEndTime(),
			Vehicle:        vehicle,
		},
		func(map[string]string, map[string]string) (*domain.Appointment, *domain.Reservation, error) {
			return appointment, reservation, nil
		},
	)
}

// appointmentFixture gives each test its own fresh dealer/technician/bay/user
// IDs and idempotency key, so tests run against the long-lived shared test DB
// without seeing each other's committed rows.
type appointmentFixture struct {
	dealerShipID   string
	technicianID   string
	serviceBayID   string
	userID         string
	idempotencyKey string
	startTime      time.Time
	duration       time.Duration
}

func newAppointmentFixture(t *testing.T) appointmentFixture {
	t.Helper()

	return appointmentFixture{
		dealerShipID:   testutils.RandomID("dealer"),
		technicianID:   testutils.RandomID("tech"),
		serviceBayID:   testutils.RandomID("bay"),
		userID:         testutils.RandomString(),
		idempotencyKey: testutils.RandomString(),
		startTime:      time.Now().Add(2 * time.Hour),
		duration:       30 * time.Minute,
	}
}

func (f appointmentFixture) endTime() time.Time {
	return f.startTime.Add(f.duration)
}

func (f appointmentFixture) build(t *testing.T) (*domain.Vehicle, *domain.Appointment, *domain.Reservation) {
	t.Helper()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: time.Hour,
		ReservationTTL:   5 * time.Minute,
	})

	vehicle, err := domain.NewVehicle("TOYOTA", "Camry", 2020, "ABC-123")
	require.NoError(t, err)

	builder, err := factory.NewAppointmentBuilder(context.Background(), domain.AppointmentData{
		DealerShipID: f.dealerShipID,
		UserID:       f.userID,
		ServiceType:  "oil-change",
		VehicleUUID:  vehicle.UUID(),
		StartTime:    f.startTime,
	})
	require.NoError(t, err)

	appointment, err := builder.
		WithDuration(f.duration).
		WithTechnicianID(f.technicianID).
		WithServiceBayID(f.serviceBayID).
		Build()
	require.NoError(t, err)

	reservation, err := factory.CreateReservation(appointment.UUID(), f.idempotencyKey)
	require.NoError(t, err)

	return vehicle, appointment, reservation
}

func (f appointmentFixture) params(vehicle *domain.Vehicle) domain.RequestAppointmentParams {
	return domain.RequestAppointmentParams{
		UserID:         f.userID,
		IdempotencyKey: f.idempotencyKey,
		DealerShipID:   f.dealerShipID,
		StartTime:      f.startTime,
		EndTime:        f.endTime(),
		Vehicle:        vehicle,
	}
}

// timeApprox tolerates the sub-millisecond precision Postgres TIMESTAMPTZ
// drops that Go's time.Now() carries.
var timeApprox = cmpopts.EquateApproxTime(time.Millisecond)

func TestAppointmentRepository_RequestAppointment_StoresDataCorrectly(t *testing.T) {
	t.Parallel()
	pool := testutils.NewDB()
	repo := appointmentsdb.NewAppointmentRepository(pool)
	f := newAppointmentFixture(t)
	vehicle, appointment, reservation := f.build(t)

	resultUUID, err := repo.RequestAppointment(
		context.Background(),
		f.params(vehicle),
		func(map[string]string, map[string]string) (*domain.Appointment, *domain.Reservation, error) {
			return appointment, reservation, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, reservation.UUID(), resultUUID)

	q := dbmodels.New(pool)
	ctx := context.Background()

	gotAppointment, err := q.GetAppointmentByUUID(ctx, appointment.UUID())
	require.NoError(t, err)
	wantAppointment := dbmodels.AppointmentsAppointment{
		AppointmentUuid:  appointment.UUID(),
		DealerShipID:     f.dealerShipID,
		ServiceBayID:     f.serviceBayID,
		TechnicianID:     f.technicianID,
		UserID:           f.userID,
		ServiceType:      "oil-change",
		StartTime:        f.startTime,
		EstimatedEndTime: f.endTime(),
		Status:           domain.AppointmentStatusPending,
		VehicleUuid:      vehicle.UUID(),
	}
	if diff := cmp.Diff(wantAppointment, gotAppointment, timeApprox); diff != "" {
		t.Errorf("appointment mismatch (-want +got):\n%s", diff)
	}

	gotReservation, err := q.GetReservationByUUID(ctx, reservation.UUID())
	require.NoError(t, err)
	wantReservation := dbmodels.AppointmentsReservation{
		ReservationUuid: reservation.UUID(),
		AppointmentUuid: appointment.UUID(),
		CreatedAt:       reservation.CreatedAt(),
		ExpiredAt:       reservation.ExpiredAt(),
		IdempotencyKey:  f.idempotencyKey,
	}
	if diff := cmp.Diff(wantReservation, gotReservation, timeApprox); diff != "" {
		t.Errorf("reservation mismatch (-want +got):\n%s", diff)
	}

	gotVehicle, err := q.GetVehicleByUUID(ctx, vehicle.UUID())
	require.NoError(t, err)
	wantVehicle := dbmodels.AppointmentsVehicle{
		VehicleUuid: vehicle.UUID(),
		UserID:      f.userID,
		MakeCode:    "TOYOTA",
		ModelName:   "Camry",
		ModelYear:   2020,
		LicensePlate: func() *string {
			lp := "ABC-123"
			return &lp
		}(),
	}
	if diff := cmp.Diff(wantVehicle, gotVehicle); diff != "" {
		t.Errorf("vehicle mismatch (-want +got):\n%s", diff)
	}
}

func TestAppointmentRepository_RequestAppointment_IdempotencyKey_ReturnsOriginalReservation(t *testing.T) {
	t.Parallel()
	pool := testutils.NewDB()
	repo := appointmentsdb.NewAppointmentRepository(pool)
	f := newAppointmentFixture(t)
	vehicle, appointment, reservation := f.build(t)

	_, found, err := repo.FindReservationByIdempotencyKey(context.Background(), f.idempotencyKey)
	require.NoError(t, err)
	assert.False(t, found, "idempotency key must not resolve before the first booking exists")

	firstUUID, err := repo.RequestAppointment(
		context.Background(),
		f.params(vehicle),
		func(map[string]string, map[string]string) (*domain.Appointment, *domain.Reservation, error) {
			return appointment, reservation, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, reservation.UUID(), firstUUID)

	foundUUID, found, err := repo.FindReservationByIdempotencyKey(context.Background(), f.idempotencyKey)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, firstUUID, foundUUID)

	replayUUID, err := repo.RequestAppointment(
		context.Background(),
		f.params(vehicle),
		func(map[string]string, map[string]string) (*domain.Appointment, *domain.Reservation, error) {
			t.Fatal("createFn should not run on an idempotency-key replay")
			return nil, nil, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, firstUUID, replayUUID)
}

// concurrentBookers is deliberately more than 2: it proves SSI rejects every
// loser regardless of how many transactions pile up on the same predicate
// lock, not just a single head-to-head pair.
const concurrentBookers = 5

// TestAppointmentRepository_RequestAppointment_RaceCondition_OnlyOneBookerWins
// deliberately reuses the same dealer/technician/bay/window across several
// concurrent bookers — the one place in this suite that violates the "unique
// technician/bay IDs" convention, because forcing the collision is the point.
func TestAppointmentRepository_RequestAppointment_RaceCondition_OnlyOneBookerWins(t *testing.T) {
	t.Parallel()
	pool := testutils.NewDB()
	repo := appointmentsdb.NewAppointmentRepository(pool)

	shared := newAppointmentFixture(t)

	type booking struct {
		fixture     appointmentFixture
		vehicle     *domain.Vehicle
		appointment *domain.Appointment
		reservation *domain.Reservation
	}

	bookings := make([]booking, concurrentBookers)
	for i := range bookings {
		f := shared
		f.userID = testutils.RandomString()
		f.idempotencyKey = testutils.RandomString()
		vehicle, appointment, reservation := f.build(t)
		bookings[i] = booking{fixture: f, vehicle: vehicle, appointment: appointment, reservation: reservation}
	}

	book := func(b booking) (domain.ReservationUUID, error) {
		return repo.RequestAppointment(
			context.Background(),
			b.fixture.params(b.vehicle),
			func(busyServiceBayIDs, busyTechnicianIDs map[string]string) (*domain.Appointment, *domain.Reservation, error) {
				if _, busy := busyTechnicianIDs[b.fixture.technicianID]; busy {
					return nil, nil, common.NewConflictError("no-availability", "technician is already booked for this window")
				}
				if _, busy := busyServiceBayIDs[b.fixture.serviceBayID]; busy {
					return nil, nil, common.NewConflictError("no-availability", "service bay is already booked for this window")
				}
				return b.appointment, b.reservation, nil
			},
		)
	}

	var wg sync.WaitGroup
	results := make([]struct {
		uuid domain.ReservationUUID
		err  error
	}, concurrentBookers)

	wg.Add(concurrentBookers)
	for i, b := range bookings {
		go func(i int, b booking) {
			defer wg.Done()
			results[i].uuid, results[i].err = book(b)
		}(i, b)
	}
	wg.Wait()

	successes, failures := 0, 0
	for _, result := range results {
		if result.err == nil {
			successes++
		} else {
			failures++
		}
	}
	assert.Equal(t, 1, successes, "exactly one concurrent booking should win")
	assert.Equal(t, concurrentBookers-1, failures, "every other concurrent booking should lose")

	busyRows, err := dbmodels.New(pool).GetBusyServiceBaysAndTechnicians(context.Background(), dbmodels.GetBusyServiceBaysAndTechniciansParams{
		DealerShipID: shared.dealerShipID,
		StartTime:    shared.startTime,
		EndTime:      shared.endTime(),
	})
	require.NoError(t, err)
	require.Len(t, busyRows, 1, "only one appointment should have been persisted for this technician/bay/window")
	assert.Equal(t, shared.technicianID, busyRows[0].TechnicianID)
	assert.Equal(t, shared.serviceBayID, busyRows[0].ServiceBayID)
}
