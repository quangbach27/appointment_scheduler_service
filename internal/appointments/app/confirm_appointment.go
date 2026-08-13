package app

import (
	"context"

	"scheduler/internal/appointments/domain"
)

type ConfirmAppointment struct {
	UserID          string
	ReservationUUID domain.ReservationUUID
}

func (s *Service) ConfirmAppointment(ctx context.Context, cmd ConfirmAppointment) (domain.AppointmentUUID, error) {
	return s.appointmentRepo.ConfirmAppointment(ctx, cmd.UserID, cmd.ReservationUUID)
}
