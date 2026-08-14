package app

import (
	"context"

	"scheduler/internal/appointments/domain"
)

type CancelBooking struct {
	UserID          string
	AppointmentUUID domain.AppointmentUUID
}

func (s *Service) CancelBooking(ctx context.Context, cmd CancelBooking) error {
	return s.appointmentRepo.CancelAppointment(ctx, cmd.UserID, cmd.AppointmentUUID)
}
