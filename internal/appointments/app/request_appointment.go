package app

import (
	"context"
	"time"

	"scheduler/internal/appointments/domain"
)

type ServiceCatalogEntry struct {
	ServiceType            string
	DisplayName            string
	Duration               time.Duration
	RequiredCertifications []string
}

type CatalogService interface {
	GetServiceEntry(ctx context.Context, serviceType string) (ServiceCatalogEntry, error)
}

type RequestBooking struct{}

func (s *Service) RequestBooking(
	ctx context.Context,
	cmd RequestBooking,
) (domain.ReservationUUID, error) {
	return domain.ReservationUUID{}, nil
}
