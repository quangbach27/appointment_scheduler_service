// Package app contains the command
package app

import (
	"scheduler/internal/appointments/domain"
)

type Service struct {
	bookingFactory  *domain.BookingFactory
	appointmentRepo domain.AppointmentRepository

	catalogService   CatalogService
	workforceService WorkforceService
}

func NewService(
	bookingFactory *domain.BookingFactory,
	appointmentRepo domain.AppointmentRepository,
	catalogService CatalogService,
	workforceService WorkforceService,
) *Service {
	if bookingFactory == nil {
		panic("bookingFactory can't be nil")
	}

	if appointmentRepo == nil {
		panic("appointmentRepo can't be nil")
	}

	if catalogService == nil {
		panic("catalogService can't be nil")
	}

	if workforceService == nil {
		panic("workforceService can't be nil")
	}

	return &Service{
		bookingFactory:   bookingFactory,
		appointmentRepo:  appointmentRepo,
		catalogService:   catalogService,
		workforceService: workforceService,
	}
}
