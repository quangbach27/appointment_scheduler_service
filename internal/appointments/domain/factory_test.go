package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
)

func TestAppointmentFactory_CreateAppointment(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: 2 * time.Hour,
	})

	startTime := time.Now().Add(3 * time.Hour)
	builder, err := factory.NewAppointmentBuilder(context.Background(), domain.AppointmentData{
		DealerShipID: "dealer-123",
		UserID:       "user-42",
		ServiceType:  "oil-change",
		VehicleUUID:  domain.VehicleUUID{UUID: common.NewUUIDv7()},
		StartTime:    startTime,
	})
	require.NoError(t, err)

	appointment, err := builder.
		WithDuration(45 * time.Minute).
		WithTechnicianID("tech-9").
		WithServiceBayID("bay-5").
		Build()

	require.NoError(t, err)
	require.NotNil(t, appointment)
	assert.Equal(t, domain.AppointmentStatusPending, appointment.Status())
	assert.False(t, appointment.IsCancelled())
	assert.False(t, appointment.IsExpired())

	details := appointment.Details()
	assert.Equal(t, "dealer-123", details.DealerShipID())
	assert.Equal(t, "bay-5", details.ServiceBayID())
	assert.Equal(t, "tech-9", details.TechnicianID())
	assert.Equal(t, "user-42", details.UserID())
	assert.Equal(t, "oil-change", details.ServiceType())
	assert.Equal(t, startTime, details.StartTime())
	assert.Equal(t, startTime.Add(45*time.Minute), details.EstimatedEndTime())
}

func TestBookingFactory_ValidateStartTime(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: 2 * time.Hour,
		MaxStartLeadTime: 7 * 24 * time.Hour,
	})

	tests := []struct {
		name        string
		startTime   time.Time
		expectError bool
	}{
		{
			name:        "rejects zero start time",
			startTime:   time.Time{},
			expectError: true,
		},
		{
			name:        "rejects start time before minimum lead time",
			startTime:   time.Now().Add(time.Hour),
			expectError: true,
		},
		{
			name:      "accepts start time just past minimum lead time",
			startTime: time.Now().Add(2*time.Hour + time.Minute),
		},
		{
			name:      "accepts start time just inside the booking window",
			startTime: time.Now().Add(7*24*time.Hour - time.Minute),
		},
		{
			name:        "rejects start time past the booking window",
			startTime:   time.Now().Add(7*24*time.Hour + time.Minute),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := factory.ValidateStartTime(tt.startTime)
			if !tt.expectError {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			var invalidErr common.Error
			require.ErrorAs(t, err, &invalidErr)
			assert.Equal(t, "invalid-appointment", invalidErr.ErrorSlug)
			require.Len(t, invalidErr.Details, 1)
			assert.Equal(t, "startTime", invalidErr.Details[0].EntityID)
		})
	}
}

func TestAppointmentFactory_NewAppointmentBuilder_ValidationErrors(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: 2 * time.Hour,
		MaxStartLeadTime: 7 * 24 * time.Hour,
	})

	tests := []struct {
		name          string
		data          domain.AppointmentData
		expectedSlugs []string
	}{
		{
			name: "rejects missing required fields",
			data: domain.AppointmentData{
				DealerShipID: "",
				UserID:       "",
				ServiceType:  "",
				StartTime:    time.Now().Add(3 * time.Hour),
			},
			expectedSlugs: []string{"required", "required", "required"},
		},
		{
			name: "rejects start time before minimum lead time",
			data: domain.AppointmentData{
				DealerShipID: "dealer-123",
				UserID:       "user-42",
				ServiceType:  "oil-change",
				StartTime:    time.Now().Add(30 * time.Minute),
			},
			expectedSlugs: []string{"out-of-range"},
		},
		{
			name: "rejects start time beyond maximum lead time",
			data: domain.AppointmentData{
				DealerShipID: "dealer-123",
				UserID:       "user-42",
				ServiceType:  "oil-change",
				StartTime:    time.Now().Add(8 * 24 * time.Hour),
			},
			expectedSlugs: []string{"out-of-range"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := factory.NewAppointmentBuilder(context.Background(), tt.data)
			require.Error(t, err)
			var invalidErr common.Error
			require.ErrorAs(t, err, &invalidErr)
			assert.Equal(t, "invalid-appointment", invalidErr.ErrorSlug)
			assert.Equal(t, "invalid appointment input", invalidErr.PublicError)

			require.Len(t, invalidErr.Details, len(tt.expectedSlugs))
			for i, expectedSlug := range tt.expectedSlugs {
				assert.Equal(t, expectedSlug, invalidErr.Details[i].ErrorSlug)
			}
		})
	}
}

func TestAppointmentBuilder_Build_ValidationErrors(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: 2 * time.Hour,
		MaxStartLeadTime: 7 * 24 * time.Hour,
	})

	newBuilder := func(t *testing.T) *domain.AppointmentBuilder {
		t.Helper()

		builder, err := factory.NewAppointmentBuilder(context.Background(), domain.AppointmentData{
			DealerShipID: "dealer-123",
			UserID:       "user-42",
			ServiceType:  "oil-change",
			StartTime:    time.Now().Add(3 * time.Hour),
		})
		require.NoError(t, err)
		return builder
	}

	tests := []struct {
		name          string
		build         func(t *testing.T) (*domain.Appointment, error)
		expectedSlugs []string
	}{
		{
			name: "rejects missing technicianID and serviceBayID",
			build: func(t *testing.T) (*domain.Appointment, error) {
				return newBuilder(t).WithDuration(30 * time.Minute).Build()
			},
			expectedSlugs: []string{"required", "required"},
		},
		{
			name: "rejects invalid duration",
			build: func(t *testing.T) (*domain.Appointment, error) {
				return newBuilder(t).
					WithDuration(0).
					WithTechnicianID("tech-9").
					WithServiceBayID("bay-5").
					Build()
			},
			expectedSlugs: []string{"out-of-range"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.build(t)
			require.Error(t, err)
			var invalidErr common.Error
			require.ErrorAs(t, err, &invalidErr)
			assert.Equal(t, "invalid-appointment", invalidErr.ErrorSlug)
			assert.Equal(t, "invalid appointment input", invalidErr.PublicError)

			require.Len(t, invalidErr.Details, len(tt.expectedSlugs))
			for i, expectedSlug := range tt.expectedSlugs {
				assert.Equal(t, expectedSlug, invalidErr.Details[i].ErrorSlug)
			}
		})
	}
}

func TestFactory_CreateReservation(t *testing.T) {
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

func TestFactory_CreateReservation_RequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{})

	reservation, err := factory.CreateReservation(domain.AppointmentUUID{UUID: common.NewUUIDv7()}, "  ")
	require.Error(t, err)
	assert.Nil(t, reservation)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "invalid-reservation", domainErr.ErrorSlug)
}
