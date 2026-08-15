package domain

import (
	"context"
	"strings"
	"time"

	"scheduler/internal/common"
)

var (
	defaultReservationTTL   = 5 * time.Minute
	defaultMinStartLeadTime = 1 * time.Hour
	defaultMaxStartLeadTime = 30 * 24 * time.Hour
)

// BookingFactory constructs Appointment instances in the Pending state and the
// Reservation tokens that guard them, enforcing the bookable start-time window
// (minimum lead time to maximum lead time) and reservation TTL respectively.
type BookingFactory struct {
	config BookingFactoryConfig
}

type BookingFactoryConfig struct {
	MinStartLeadTime time.Duration
	MaxStartLeadTime time.Duration
	ReservationTTL   time.Duration
}

func NewBookingFactory(config BookingFactoryConfig) *BookingFactory {
	if config.MinStartLeadTime <= 0 {
		config.MinStartLeadTime = defaultMinStartLeadTime
	}

	if config.MaxStartLeadTime <= 0 {
		config.MaxStartLeadTime = defaultMaxStartLeadTime
	}

	if config.ReservationTTL <= 0 {
		config.ReservationTTL = defaultReservationTTL
	}

	return &BookingFactory{
		config: config,
	}
}

type AppointmentData struct {
	DealerShipID string
	UserID       string
	ServiceType  string
	VehicleUUID  VehicleUUID
	StartTime    time.Time
}

// AppointmentBuilder assembles an Appointment across the two points in
// RequestBooking where its data becomes available: NewAppointmentBuilder
// validates the caller-supplied fields up front, before any external call;
// WithDuration is filled in once the catalog responds; WithTechnicianID and
// WithServiceBayID are filled in from createFn once availability is
// resolved, and Build validates and constructs the Appointment.
type AppointmentBuilder struct {
	data              AppointmentData
	estimatedDuration time.Duration
	technicianID      string
	serviceBayID      string
}

// NewAppointmentBuilder validates DealerShipID, UserID, ServiceType, and
// StartTime, and fails fast on bad input before any catalog or workforce
// call is made.
func (f *BookingFactory) NewAppointmentBuilder(ctx context.Context, data AppointmentData) (*AppointmentBuilder, error) {
	errDetails := []common.ErrorDetails{}
	if data.DealerShipID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "dealerShipID",
			ErrorSlug:  "required",
			Message:    "dealerShipID is required",
		})
	}

	if data.UserID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "userID",
			ErrorSlug:  "required",
			Message:    "userID is required",
		})
	}

	if data.ServiceType == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "serviceType",
			ErrorSlug:  "required",
			Message:    "serviceType is required",
		})
	}

	if detail := f.startTimeErrorDetail(data.StartTime); detail != nil {
		errDetails = append(errDetails, *detail)
	}

	if len(errDetails) > 0 {
		return nil, common.NewInvalidInputError("invalid-appointment", "invalid appointment input").WithDetails(errDetails)
	}

	return &AppointmentBuilder{data: data}, nil
}

func (b *AppointmentBuilder) WithDuration(estimatedDuration time.Duration) *AppointmentBuilder {
	b.estimatedDuration = estimatedDuration
	return b
}

func (b *AppointmentBuilder) WithTechnicianID(technicianID string) *AppointmentBuilder {
	b.technicianID = technicianID
	return b
}

func (b *AppointmentBuilder) WithServiceBayID(serviceBayID string) *AppointmentBuilder {
	b.serviceBayID = serviceBayID
	return b
}

func (b *AppointmentBuilder) Build() (*Appointment, error) {
	errDetails := []common.ErrorDetails{}
	if b.technicianID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "technicianID",
			ErrorSlug:  "required",
			Message:    "technicianID is required",
		})
	}

	if b.serviceBayID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "serviceBayID",
			ErrorSlug:  "required",
			Message:    "serviceBayID is required",
		})
	}

	if b.estimatedDuration <= 0 {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "estimatedDuration",
			ErrorSlug:  "out-of-range",
			Message:    "estimatedDuration must be greater than 0",
		})
	}

	if len(errDetails) > 0 {
		return nil, common.NewInvalidInputError("invalid-appointment", "invalid appointment input").WithDetails(errDetails)
	}

	return &Appointment{
		uuid:   AppointmentUUID{UUID: common.NewUUIDv7()},
		status: AppointmentStatusPending,
		details: AppointmentDetails{
			dealerShipID:     b.data.DealerShipID,
			serviceBayID:     b.serviceBayID,
			technicianID:     b.technicianID,
			userID:           b.data.UserID,
			serviceType:      b.data.ServiceType,
			vehicleUUID:      b.data.VehicleUUID,
			startTime:        b.data.StartTime,
			estimatedEndTime: b.data.StartTime.Add(b.estimatedDuration),
		},
	}, nil
}

// ValidateStartTime reports whether a start time falls inside the bookable
// window, so callers can reject early instead of after external lookups.
// CreateAppointment applies the same rule, so the factory stays self-guarding.
func (f *BookingFactory) ValidateStartTime(startTime time.Time) error {
	detail := f.startTimeErrorDetail(startTime)
	if detail == nil {
		return nil
	}

	return common.NewInvalidInputError("invalid-appointment", "invalid appointment input").
		WithDetails([]common.ErrorDetails{*detail})
}

// startTimeErrorDetail is the single definition of the bookable-window rule:
// a start time must be present, no sooner than the minimum lead time, and no
// further out than the maximum lead time.
func (f *BookingFactory) startTimeErrorDetail(startTime time.Time) *common.ErrorDetails {
	now := time.Now()

	switch {
	case startTime.IsZero():
		return &common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "startTime",
			ErrorSlug:  "required",
			Message:    "startTime is required",
		}
	case startTime.Before(now.Add(f.config.MinStartLeadTime)):
		return &common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "startTime",
			ErrorSlug:  "out-of-range",
			Message:    "startTime must satisfy minimum lead time",
		}
	case startTime.After(now.Add(f.config.MaxStartLeadTime)):
		return &common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "startTime",
			ErrorSlug:  "out-of-range",
			Message:    "startTime must be within the booking window",
		}
	default:
		return nil
	}
}

func (f *BookingFactory) CreateReservation(appointmentUUID AppointmentUUID, idempotencyKey string) (*Reservation, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, common.NewInvalidInputError("invalid-reservation", "idempotencyKey is required")
	}

	now := time.Now()

	return &Reservation{
		uuid:            ReservationUUID{UUID: common.NewUUIDv7()},
		appointmentUUID: appointmentUUID,
		createdAt:       now,
		expiredAt:       now.Add(f.config.ReservationTTL),
		idempotencyKey:  idempotencyKey,
	}, nil
}
