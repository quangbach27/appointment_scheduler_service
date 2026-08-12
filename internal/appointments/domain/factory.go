package domain

import (
	"time"

	"scheduler/internal/common"
)

var (
	defaultReservationTTL   = 5 * time.Minute
	defaultMinStartLeadTime = 1 * time.Hour
)

// BookingFactory constructs Appointment instances in the Pending state and the
// Reservation tokens that guard them, enforcing minimum start-time lead time
// and reservation TTL respectively.
type BookingFactory struct {
	minStartLeadTime time.Duration
	reservationTTL   time.Duration
}

type BookingFactoryConfig struct {
	MinStartLeadTime time.Duration
	ReservationTTL   time.Duration
}

func NewBookingFactory(config BookingFactoryConfig) *BookingFactory {
	minStartLeadTime := config.MinStartLeadTime
	if minStartLeadTime <= 0 {
		minStartLeadTime = defaultMinStartLeadTime
	}

	reservationTTL := config.ReservationTTL
	if reservationTTL <= 0 {
		reservationTTL = defaultReservationTTL
	}

	return &BookingFactory{
		minStartLeadTime: minStartLeadTime,
		reservationTTL:   reservationTTL,
	}
}

type AppointmentData struct {
	DealerShipID string
	ServiceBayID string
	TechnicianID string
	UserID       string

	ServiceType string
	VehicleUUID VehicleUUID

	StartTime         time.Time
	EstimatedDuration time.Duration
}

func (f *BookingFactory) CreateAppointment(data AppointmentData) (*Appointment, error) {
	details, err := f.createAppointmentDetails(data)
	if err != nil {
		return nil, err
	}

	return &Appointment{
		uuid:    AppointmentUUID{UUID: common.NewUUIDv7()},
		status:  AppointmentStatusPending,
		details: details,
	}, nil
}

func (f *BookingFactory) createAppointmentDetails(data AppointmentData) (AppointmentDetails, error) {
	errDetails := []common.ErrorDetails{}
	now := time.Now()

	if data.DealerShipID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "dealerShipID",
			ErrorSlug:  "required",
			Message:    "dealerShipID is required",
		})
	}

	if data.ServiceBayID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "serviceBayID",
			ErrorSlug:  "required",
			Message:    "serviceBayID is required",
		})
	}

	if data.TechnicianID == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "technicianID",
			ErrorSlug:  "required",
			Message:    "technicianID is required",
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

	estimatedDuration := data.EstimatedDuration
	if estimatedDuration <= 0 {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "estimatedDuration",
			ErrorSlug:  "out-of-range",
			Message:    "estimatedDuration must be greater than 0",
		})
	}

	if data.StartTime.IsZero() {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "startTime",
			ErrorSlug:  "required",
			Message:    "startTime is required",
		})
	} else if data.StartTime.Before(now.Add(f.minStartLeadTime)) {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Appointment",
			EntityID:   "startTime",
			ErrorSlug:  "out-of-range",
			Message:    "startTime must satisfy minimum lead time",
		})
	}

	if len(errDetails) > 0 {
		return AppointmentDetails{}, common.NewInvalidInputError("invalid-appointment", "invalid appointment input").WithDetails(errDetails)
	}

	return AppointmentDetails{
		dealerShipID:     data.DealerShipID,
		serviceBayID:     data.ServiceBayID,
		technicianID:     data.TechnicianID,
		userID:           data.UserID,
		serviceType:      data.ServiceType,
		vehicleUUID:      data.VehicleUUID,
		startTime:        data.StartTime,
		estimatedEndTime: data.StartTime.Add(data.EstimatedDuration),
	}, nil
}

func (f *BookingFactory) CreateReservation(appointmentUUID AppointmentUUID) *Reservation {
	now := time.Now()

	return &Reservation{
		uuid:            ReservationUUID{UUID: common.NewUUIDv7()},
		appointmentUUID: appointmentUUID,
		createdAt:       now,
		expiredAt:       now.Add(f.reservationTTL),
	}
}
