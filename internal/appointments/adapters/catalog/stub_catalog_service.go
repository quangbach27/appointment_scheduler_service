// Package catalog adapts the external service catalog to app.CatalogService.
package catalog

import (
	"context"
	"time"

	"scheduler/internal/appointments/app"
	"scheduler/internal/common"
)

// StubCatalogService serves a fixed service catalog, standing in for the real
// catalog service until it exists. Component tests use it so booking flows can
// run without a real upstream.
type StubCatalogService struct {
	entries map[string]app.ServiceCatalogEntry
}

var _ app.CatalogService = (*StubCatalogService)(nil)

func NewStubCatalogService() *StubCatalogService {
	return &StubCatalogService{
		entries: map[string]app.ServiceCatalogEntry{
			"oil-change": {
				ServiceType:            "oil-change",
				DisplayName:            "Oil Change",
				Duration:               time.Hour,
				RequiredCertifications: []string{},
			},
			"brake-service": {
				ServiceType:            "brake-service",
				DisplayName:            "Brake Service",
				Duration:               2 * time.Hour,
				RequiredCertifications: []string{"brake-certified"},
			},
			"diagnostics": {
				ServiceType:            "diagnostics",
				DisplayName:            "Full Diagnostics",
				Duration:               90 * time.Minute,
				RequiredCertifications: []string{"diagnostics-certified"},
			},
		},
	}
}

func (s *StubCatalogService) GetServiceEntry(_ context.Context, serviceType string) (app.ServiceCatalogEntry, error) {
	entry, ok := s.entries[serviceType]
	if !ok {
		return app.ServiceCatalogEntry{}, common.NewNotFoundError(
			"service-type-not-found",
			"unknown service type %q",
			serviceType,
		)
	}

	return entry, nil
}
