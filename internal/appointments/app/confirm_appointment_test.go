package app_test

import (
	"context"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"scheduler/internal/appointments/app"
	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"
)

type fakeAppointmentRepository struct {
	confirmUserID          string
	confirmReservationUUID domain.ReservationUUID

	confirmResult domain.AppointmentUUID
	confirmErr    error
}

func (f *fakeAppointmentRepository) RequestAppointment(context.Context, string, *domain.Vehicle, *domain.Appointment, *domain.Reservation) (domain.ReservationUUID, error) {
	return domain.ReservationUUID{}, nil
}

func (f *fakeAppointmentRepository) ConfirmAppointment(ctx context.Context, userID string, reservationUUID domain.ReservationUUID) (domain.AppointmentUUID, error) {
	f.confirmUserID = userID
	f.confirmReservationUUID = reservationUUID
	return f.confirmResult, f.confirmErr
}

func (f *fakeAppointmentRepository) CancelAppointment(context.Context, string, domain.AppointmentUUID) error {
	return nil
}

type fakeCatalogService struct{}

func (fakeCatalogService) GetServiceEntry(context.Context, string) (app.ServiceCatalogEntry, error) {
	return app.ServiceCatalogEntry{}, nil
}

func TestService_ConfirmAppointment_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	wantAppointmentUUID := domain.AppointmentUUID{UUID: common.NewUUIDv7()}
	repo := &fakeAppointmentRepository{confirmResult: wantAppointmentUUID}
	service := app.NewService(domain.NewBookingFactory(domain.BookingFactoryConfig{}), repo, fakeCatalogService{})

	reservationUUID := domain.ReservationUUID{UUID: common.NewUUIDv7()}
	got, err := service.ConfirmAppointment(context.Background(), app.ConfirmAppointment{
		UserID:          "user-1",
		ReservationUUID: reservationUUID,
	})

	require.NoError(t, err)
	assert.Equal(t, wantAppointmentUUID, got)
	assert.Equal(t, "user-1", repo.confirmUserID)
	assert.Equal(t, reservationUUID, repo.confirmReservationUUID)
}

func TestService_ConfirmAppointment_ReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := common.NewNotFoundError("reservation-not-found", "reservation not found")
	repo := &fakeAppointmentRepository{confirmErr: wantErr}
	service := app.NewService(domain.NewBookingFactory(domain.BookingFactoryConfig{}), repo, fakeCatalogService{})

	_, err := service.ConfirmAppointment(context.Background(), app.ConfirmAppointment{
		UserID:          "user-1",
		ReservationUUID: domain.ReservationUUID{UUID: common.NewUUIDv7()},
	})

	require.Error(t, err)
	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "reservation-not-found", domainErr.ErrorSlug)
}
