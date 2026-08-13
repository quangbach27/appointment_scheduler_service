// Package app contains the command
package app

import (
	"scheduler/internal/appointments/domain"
)

type Service struct {
	bookingFactory  *domain.BookingFactory
	appointmentRepo domain.AppointmentRepository

	catalogService CatalogService
}

func NewService(
	bookingFactory *domain.BookingFactory,
	appointmentRepo domain.AppointmentRepository,
	catalogService CatalogService,
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

	return &Service{
		bookingFactory:  bookingFactory,
		catalogService:  catalogService,
		appointmentRepo: appointmentRepo,
	}
}
