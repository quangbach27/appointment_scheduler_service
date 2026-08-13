package app

import "context"

type CancelAppointment struct{}

func (s *Service) CancelAppointment(ctx context.Context, cmd CancelAppointment) error {
	return nil
}
