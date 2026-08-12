package domain

import "time"

// UnmarshalReservation hydrates Reservation from trusted persistence data.
func UnmarshalReservation(
	uuid ReservationUUID,
	appointmentUUID AppointmentUUID,
	createdAt time.Time,
	expiredAt time.Time,
) *Reservation {
	return &Reservation{
		uuid:            uuid,
		appointmentUUID: appointmentUUID,
		createdAt:       createdAt,
		expiredAt:       expiredAt,
	}
}

func UnmarshalAppointment(
	uuid AppointmentUUID,
	dealerShipID string,
	serviceBayID string,
	technicianID string,
	userID string,
	serviceType string,
	vehicleUUID VehicleUUID,
	startTime time.Time,
	estimatedEndTime time.Time,
	status AppointmentStatus,
) *Appointment {
	details := UnmarshalAppointmentDetails(
		dealerShipID,
		serviceBayID,
		technicianID,
		userID,
		serviceType,
		vehicleUUID,
		startTime,
		estimatedEndTime,
	)

	return &Appointment{
		uuid:    uuid,
		details: details,
		status:  status,
	}
}

func UnmarshalAppointmentDetails(
	dealerShipID string,
	serviceBayID string,
	technicianID string,
	userID string,
	serviceType string,
	vehicleUUID VehicleUUID,
	startTime time.Time,
	estimatedEndTime time.Time,
) AppointmentDetails {
	return AppointmentDetails{
		dealerShipID:     dealerShipID,
		serviceBayID:     serviceBayID,
		technicianID:     technicianID,
		userID:           userID,
		serviceType:      serviceType,
		vehicleUUID:      vehicleUUID,
		startTime:        startTime,
		estimatedEndTime: estimatedEndTime,
	}
}
