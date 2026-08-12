package domain

import "time"

type AppointmentDetails struct {
	dealerShipID     string
	serviceBayID     string
	technicianID     string
	userID           string
	serviceType      string
	vehicleUUID      VehicleUUID
	startTime        time.Time
	estimatedEndTime time.Time
}

func (a AppointmentDetails) IsZero() bool {
	return a == AppointmentDetails{}
}

func (a AppointmentDetails) DealerShipID() string {
	return a.dealerShipID
}

func (a AppointmentDetails) ServiceBayID() string {
	return a.serviceBayID
}

func (a AppointmentDetails) TechnicianID() string {
	return a.technicianID
}

func (a AppointmentDetails) UserID() string {
	return a.userID
}

func (a AppointmentDetails) ServiceType() string {
	return a.serviceType
}

func (a AppointmentDetails) VehicleUUID() VehicleUUID {
	return a.vehicleUUID
}

func (a AppointmentDetails) StartTime() time.Time {
	return a.startTime
}

func (a AppointmentDetails) EstimatedEndTime() time.Time {
	return a.estimatedEndTime
}
