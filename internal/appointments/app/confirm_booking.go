package app

import (
	"context"

	"scheduler/internal/appointments/domain"
)

type ConfirmBooking struct {
	UserID          string
	ReservationUUID domain.ReservationUUID
}

func (s *Service) ConfirmBooking(ctx context.Context, cmd ConfirmBooking) (domain.AppointmentUUID, error) {
	return s.appointmentRepo.ConfirmAppointment(ctx, cmd.UserID, cmd.ReservationUUID)
}
