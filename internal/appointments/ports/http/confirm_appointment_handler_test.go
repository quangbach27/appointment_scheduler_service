package http_test

import (
	"context"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"scheduler/internal/appointments/app"
	"scheduler/internal/appointments/domain"
	httpport "scheduler/internal/appointments/ports/http"
	"scheduler/internal/common"
)

type fakeAppointmentRepository struct {
	called bool

	confirmResult domain.AppointmentUUID
	confirmErr    error
}

func (f *fakeAppointmentRepository) RequestAppointment(context.Context, string, *domain.Vehicle, *domain.Appointment, *domain.Reservation) (domain.ReservationUUID, error) {
	return domain.ReservationUUID{}, nil
}

func (f *fakeAppointmentRepository) ConfirmAppointment(context.Context, string, domain.ReservationUUID) (domain.AppointmentUUID, error) {
	f.called = true
	return f.confirmResult, f.confirmErr
}

func (f *fakeAppointmentRepository) CancelAppointment(context.Context, string, domain.AppointmentUUID) error {
	return nil
}

type fakeCatalogService struct{}

func (fakeCatalogService) GetServiceEntry(context.Context, string) (app.ServiceCatalogEntry, error) {
	return app.ServiceCatalogEntry{}, nil
}

func newTestHandlers(repo *fakeAppointmentRepository) httpport.Handlers {
	service := app.NewService(domain.NewBookingFactory(domain.BookingFactoryConfig{}), repo, fakeCatalogService{})
	return httpport.NewHandlers(service)
}

func TestHandlers_ConfirmAppointment_Success(t *testing.T) {
	t.Parallel()

	wantAppointmentUUID := domain.AppointmentUUID{UUID: common.NewUUIDv7()}
	repo := &fakeAppointmentRepository{confirmResult: wantAppointmentUUID}
	handlers := newTestHandlers(repo)

	reservationUUID := common.NewUUIDv7()
	resp, err := handlers.ConfirmAppointment(context.Background(), httpport.ConfirmAppointmentRequestObject{
		ReservationUuid: openapi_types.UUID(reservationUUID),
		Params:          httpport.ConfirmAppointmentParams{XUserId: "user-1"},
	})

	require.NoError(t, err)
	require.True(t, repo.called)

	successResp, ok := resp.(httpport.ConfirmAppointment200JSONResponse)
	require.True(t, ok, "expected a 200 JSON response, got %T", resp)
	assert.Equal(t, openapi_types.UUID(wantAppointmentUUID.UUID), successResp.AppointmentUuid)
	assert.Equal(t, httpport.Confirmed, successResp.Status)
}

func TestHandlers_ConfirmAppointment_RepositoryErrorPassesThrough(t *testing.T) {
	t.Parallel()

	wantErr := common.NewExpiredError("reservation-expired", "reservation expired")
	repo := &fakeAppointmentRepository{confirmErr: wantErr}
	handlers := newTestHandlers(repo)

	resp, err := handlers.ConfirmAppointment(context.Background(), httpport.ConfirmAppointmentRequestObject{
		ReservationUuid: openapi_types.UUID(common.NewUUIDv7()),
		Params:          httpport.ConfirmAppointmentParams{XUserId: "user-1"},
	})

	require.Error(t, err)
	assert.Nil(t, resp)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "reservation-expired", domainErr.ErrorSlug)
}

func TestHandlers_ConfirmAppointment_MissingUserIDReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	repo := &fakeAppointmentRepository{}
	handlers := newTestHandlers(repo)

	resp, err := handlers.ConfirmAppointment(context.Background(), httpport.ConfirmAppointmentRequestObject{
		ReservationUuid: openapi_types.UUID(common.NewUUIDv7()),
		Params:          httpport.ConfirmAppointmentParams{XUserId: "   "},
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.False(t, repo.called)

	var domainErr common.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, 401, domainErr.HttpErrorCode)
}
