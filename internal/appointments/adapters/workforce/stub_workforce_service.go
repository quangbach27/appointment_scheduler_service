// Package workforce adapts the external leadership-management service, which
// owns technicians and service bays, to app.WorkforceService.
package workforce

import (
	"context"
	"time"

	"scheduler/internal/appointments/app"
)

// blackout is a window in which a resource is not on duty: a technician's day
// off, a service bay under maintenance.
type blackout struct {
	from time.Time
	to   time.Time
}

type technicianEntry struct {
	technician app.Technician
	blackouts  []blackout
}

type serviceBayEntry struct {
	serviceBay app.ServiceBay
	blackouts  []blackout
}

// A blackout wide enough to cover any window a caller can ask about, for the
// fixtures that model a permanently unavailable resource.
var (
	distantPast   = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	distantFuture = time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC)
)

func alwaysOff() []blackout {
	return []blackout{{from: distantPast, to: distantFuture}}
}

// StubWorkforceService serves a fixed roster, standing in for the real
// leadership-management service until it exists. Component tests use it so
// booking flows can run without a real upstream.
type StubWorkforceService struct {
	technicians map[string][]technicianEntry
	serviceBays map[string][]serviceBayEntry
}

var _ app.WorkforceService = (*StubWorkforceService)(nil)

func NewStubWorkforceService() *StubWorkforceService {
	return &StubWorkforceService{
		technicians: map[string][]technicianEntry{
			// Distinct review scores so component tests can assert the
			// highest-scored available technician wins (tech-2 over tech-1).
			"dealer-1": {
				{technician: app.Technician{ID: "tech-1", Certifications: []string{"brake-certified"}, ReviewScore: 3.5}},
				{technician: app.Technician{ID: "tech-2", Certifications: []string{"brake-certified", "diagnostics-certified"}, ReviewScore: 4.8}},
				{technician: app.Technician{ID: "tech-3", Certifications: []string{}, ReviewScore: 4.0}},
			},
			// Single technician and single bay, so component tests can exhaust
			// availability and exercise the concurrent-booking race.
			"dealer-solo": {
				{technician: app.Technician{ID: "solo-tech-1", Certifications: []string{"brake-certified"}, ReviewScore: 4.0}},
			},
			// The only technician is permanently off duty, so bookings here always
			// run out of candidates however free our own calendar is.
			"dealer-offday": {
				{
					technician: app.Technician{ID: "offday-tech-1", Certifications: []string{"brake-certified"}, ReviewScore: 4.0},
					blackouts:  alwaysOff(),
				},
			},
			"dealer-maintenance": {
				{technician: app.Technician{ID: "maintenance-tech-1", Certifications: []string{"brake-certified"}, ReviewScore: 4.0}},
			},
		},
		serviceBays: map[string][]serviceBayEntry{
			"dealer-1":      {{serviceBay: app.ServiceBay{ID: "bay-1"}}, {serviceBay: app.ServiceBay{ID: "bay-2"}}},
			"dealer-solo":   {{serviceBay: app.ServiceBay{ID: "solo-bay-1"}}},
			"dealer-offday": {{serviceBay: app.ServiceBay{ID: "offday-bay-1"}}},
			// Mirror of dealer-offday on the bay side: the only bay is permanently
			// under maintenance.
			"dealer-maintenance": {
				{serviceBay: app.ServiceBay{ID: "maintenance-bay-1"}, blackouts: alwaysOff()},
			},
		},
	}
}

func (s *StubWorkforceService) ListTechnicians(
	_ context.Context,
	query app.ListTechniciansRequest,
) ([]app.Technician, error) {
	matched := []app.Technician{}

	for _, entry := range s.technicians[query.DealerShipID] {
		if !hasAllCertifications(entry.technician, query.RequiredCertifications) {
			continue
		}

		if !onDuty(entry.blackouts, query.StartTime, query.EndTime) {
			continue
		}

		matched = append(matched, entry.technician)
	}

	return matched, nil
}

func (s *StubWorkforceService) ListServiceBays(
	_ context.Context,
	query app.ListServiceBaysRequest,
) ([]app.ServiceBay, error) {
	matched := []app.ServiceBay{}

	for _, entry := range s.serviceBays[query.DealerShipID] {
		if onDuty(entry.blackouts, query.StartTime, query.EndTime) {
			matched = append(matched, entry.serviceBay)
		}
	}

	return matched, nil
}

// onDuty reports whether no blackout overlaps the requested window. Overlap is
// half-open, matching how booked windows overlap, so a booking starting exactly
// when maintenance ends is fine.
func onDuty(blackouts []blackout, startTime, endTime time.Time) bool {
	for _, b := range blackouts {
		if b.from.Before(endTime) && b.to.After(startTime) {
			return false
		}
	}

	return true
}

func hasAllCertifications(technician app.Technician, required []string) bool {
	held := make(map[string]struct{}, len(technician.Certifications))
	for _, certification := range technician.Certifications {
		held[certification] = struct{}{}
	}

	for _, certification := range required {
		if _, ok := held[certification]; !ok {
			return false
		}
	}

	return true
}
