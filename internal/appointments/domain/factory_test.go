package domain_test

import (
	"testing"
	"time"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestAppointmentFactory_CreateAppointment(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: 2 * time.Hour,
	})

	startTime := time.Now().Add(3 * time.Hour)
	appointment, err := factory.CreateAppointment(domain.AppointmentData{
		DealerShipID:      "dealer-123",
		ServiceBayID:      "bay-5",
		TechnicianID:      "tech-9",
		UserID:            "user-42",
		ServiceType:       "oil-change",
		VehicleUUID:       domain.VehicleUUID{UUID: common.NewUUIDv7()},
		StartTime:         startTime,
		EstimatedDuration: 45 * time.Minute,
	})

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

func TestAppointmentFactory_CreateAppointment_DefaultsAreApplied(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{})

	// default MinStartLeadTime is 1 hour; 30 minutes out should be rejected.
	_, err := factory.CreateAppointment(domain.AppointmentData{
		DealerShipID:      "dealer-123",
		ServiceBayID:      "bay-5",
		TechnicianID:      "tech-9",
		UserID:            "user-42",
		ServiceType:       "oil-change",
		VehicleUUID:       domain.VehicleUUID{UUID: common.NewUUIDv7()},
		StartTime:         time.Now().Add(30 * time.Minute),
		EstimatedDuration: 30 * time.Minute,
	})
	require.Error(t, err)

	appointment, err := factory.CreateAppointment(domain.AppointmentData{
		DealerShipID:      "dealer-123",
		ServiceBayID:      "bay-5",
		TechnicianID:      "tech-9",
		UserID:            "user-42",
		ServiceType:       "oil-change",
		VehicleUUID:       domain.VehicleUUID{UUID: common.NewUUIDv7()},
		StartTime:         time.Now().Add(2 * time.Hour),
		EstimatedDuration: 30 * time.Minute,
	})
	require.NoError(t, err)
	require.NotNil(t, appointment)
}

func TestAppointmentFactory_CreateAppointment_ValidationErrors(t *testing.T) {
	t.Parallel()

	factory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: 2 * time.Hour,
	})

	tests := []struct {
		name          string
		data          domain.AppointmentData
		expectedSlugs []string
	}{
		{
			name: "rejects missing required fields",
			data: domain.AppointmentData{
				DealerShipID:      "",
				ServiceBayID:      "",
				TechnicianID:      "",
				UserID:            "",
				ServiceType:       "",
				StartTime:         time.Now().Add(3 * time.Hour),
				EstimatedDuration: 30 * time.Minute,
			},
			expectedSlugs: []string{"required", "required", "required", "required", "required"},
		},
		{
			name: "rejects invalid duration",
			data: domain.AppointmentData{
				DealerShipID:      "dealer-123",
				ServiceBayID:      "bay-5",
				TechnicianID:      "tech-9",
				UserID:            "user-42",
				ServiceType:       "oil-change",
				StartTime:         time.Now().Add(3 * time.Hour),
				EstimatedDuration: 0,
			},
			expectedSlugs: []string{"out-of-range"},
		},
		{
			name: "rejects start time before minimum lead time",
			data: domain.AppointmentData{
				DealerShipID:      "dealer-123",
				ServiceBayID:      "bay-5",
				TechnicianID:      "tech-9",
				UserID:            "user-42",
				ServiceType:       "oil-change",
				StartTime:         time.Now().Add(30 * time.Minute),
				EstimatedDuration: 30 * time.Minute,
			},
			expectedSlugs: []string{"out-of-range"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := factory.CreateAppointment(tt.data)
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
