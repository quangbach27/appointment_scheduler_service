package app

import (
	"context"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
	"scheduler/internal/common/log"
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

type Technician struct {
	ID             string
	Certifications []string
	ReviewScore    float64
}

type ServiceBay struct {
	ID string
}

type ListTechniciansRequest struct {
	DealerShipID           string
	RequiredCertifications []string
	StartTime              time.Time
	EndTime                time.Time
}

type ListServiceBaysRequest struct {
	DealerShipID string
	StartTime    time.Time
	EndTime      time.Time
}

// WorkforceService exposes the resources a dealership's leadership manages:
// which technicians hold which certifications and review score, which service
// bays exist, and which of them are on duty for a given window — a technician
// may be off that day, a service bay may be under maintenance. It knows
// nothing about who is already booked; that is decided against our own
// appointment records.
type WorkforceService interface {
	ListTechnicians(ctx context.Context, req ListTechniciansRequest) ([]Technician, error)
	ListServiceBays(ctx context.Context, req ListServiceBaysRequest) ([]ServiceBay, error)
}

type RequestBookingVehicle struct {
	MakeCode     string
	ModelName    string
	ModelYear    int
	LicensePlate string
}

type RequestBooking struct {
	UserID         string
	DealerShipID   string
	ServiceType    string
	StartTime      time.Time
	IdempotencyKey string
	Vehicle        RequestBookingVehicle
}

func (s *Service) RequestBooking(
	ctx context.Context,
	cmd RequestBooking,
) (domain.ReservationUUID, error) {
	if err := s.bookingFactory.ValidateStartTime(cmd.StartTime); err != nil {
		return domain.ReservationUUID{}, err
	}

	if reservationUUID, found, err := s.appointmentRepo.FindReservationByIdempotencyKey(
		ctx,
		cmd.IdempotencyKey,
	); err != nil {
		return domain.ReservationUUID{}, err
	} else if found {
		log.FromContext(ctx).Info("idempotent replay: fast-path hit, skipping booking",
			"idempotency_key", cmd.IdempotencyKey,
			"reservation_uuid", reservationUUID.String(),
		)
		return reservationUUID, nil
	}

	vehicle, err := domain.NewVehicle(
		cmd.Vehicle.MakeCode,
		cmd.Vehicle.ModelName,
		cmd.Vehicle.ModelYear,
		cmd.Vehicle.LicensePlate,
	)
	if err != nil {
		return domain.ReservationUUID{}, err
	}

	appointmentBuilder, err := s.bookingFactory.NewAppointmentBuilder(ctx, domain.AppointmentData{
		DealerShipID: cmd.DealerShipID,
		UserID:       cmd.UserID,
		ServiceType:  cmd.ServiceType,
		VehicleUUID:  vehicle.UUID(),
		StartTime:    cmd.StartTime,
	})
	if err != nil {
		return domain.ReservationUUID{}, err
	}

	entry, err := s.catalogService.GetServiceEntry(ctx, cmd.ServiceType)
	if err != nil {
		return domain.ReservationUUID{}, err
	}
	appointmentBuilder = appointmentBuilder.WithDuration(entry.Duration)

	endTime := cmd.StartTime.Add(entry.Duration)

	technicians, bays, err := s.listAvailabilityCandidates(
		ctx,
		cmd.DealerShipID,
		entry.RequiredCertifications,
		cmd.StartTime,
		endTime,
	)
	if err != nil {
		return domain.ReservationUUID{}, err
	}

	if len(technicians) == 0 || len(bays) == 0 {
		return domain.ReservationUUID{}, errNoAvailability()
	}

	// Highest-reviewed technician first, so the busy-set filter below picks the
	// best-scored one still available.
	sort.Slice(technicians, func(i, j int) bool {
		return technicians[i].ReviewScore > technicians[j].ReviewScore
	})

	return s.appointmentRepo.RequestAppointment(
		ctx,
		domain.RequestAppointmentParams{
			UserID:         cmd.UserID,
			IdempotencyKey: cmd.IdempotencyKey,
			DealerShipID:   cmd.DealerShipID,
			StartTime:      cmd.StartTime,
			EndTime:        endTime,
			Vehicle:        vehicle,
		},
		func(
			busyServiceBayIDs,
			busyTechnicianIDs map[string]string,
		) (*domain.Appointment, *domain.Reservation, error) {
			var technicianID string
			for _, technician := range technicians {
				if _, busy := busyTechnicianIDs[technician.ID]; !busy {
					technicianID = technician.ID
					break
				}
			}

			var serviceBayID string
			for _, bay := range bays {
				if _, busy := busyServiceBayIDs[bay.ID]; !busy {
					serviceBayID = bay.ID
					break
				}
			}

			if technicianID == "" || serviceBayID == "" {
				return nil, nil, errNoAvailability()
			}

			appointment, err := appointmentBuilder.
				WithTechnicianID(technicianID).
				WithServiceBayID(serviceBayID).
				Build()
			if err != nil {
				return nil, nil, err
			}

			reservation, err := s.bookingFactory.CreateReservation(
				appointment.UUID(),
				cmd.IdempotencyKey,
			)
			if err != nil {
				return nil, nil, err
			}

			return appointment, reservation, nil
		},
	)
}

// listAvailabilityCandidates fetches the on-duty technicians and service bays
// for the given dealership and window concurrently, since neither call
// depends on the other's result.
func (s *Service) listAvailabilityCandidates(
	ctx context.Context,
	dealerShipID string,
	requiredCertifications []string,
	startTime, endTime time.Time,
) ([]Technician, []ServiceBay, error) {
	var technicians []Technician
	var bays []ServiceBay

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		technicians, err = s.workforceService.ListTechnicians(gCtx, ListTechniciansRequest{
			DealerShipID:           dealerShipID,
			RequiredCertifications: requiredCertifications,
			StartTime:              startTime,
			EndTime:                endTime,
		})
		return err
	})

	g.Go(func() error {
		var err error
		bays, err = s.workforceService.ListServiceBays(gCtx, ListServiceBaysRequest{
			DealerShipID: dealerShipID,
			StartTime:    startTime,
			EndTime:      endTime,
		})
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	return technicians, bays, nil
}

func errNoAvailability() error {
	return common.NewConflictError(
		"no-availability",
		"no technician or service bay is available for this time, please choose another time",
	)
}
