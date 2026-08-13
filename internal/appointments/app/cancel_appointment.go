package app

import (
	"context"

	"scheduler/internal/appointments/domain"
)

type CancelAppointment struct {
	UserID          string
	AppointmentUUID domain.AppointmentUUID
}

func (s *Service) CancelAppointment(ctx context.Context, cmd CancelAppointment) error {
	return s.appointmentRepo.CancelAppointment(ctx, cmd.UserID, cmd.AppointmentUUID)
}
