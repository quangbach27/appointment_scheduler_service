package appointments

import (
	"context"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler/internal/appointments/adapters/db"
	"scheduler/internal/appointments/app"
	"scheduler/internal/appointments/domain"
	httpPort "scheduler/internal/appointments/ports/http"
	"scheduler/internal/common"
	"scheduler/internal/common/config"
	"scheduler/internal/common/module/contracts"
)

type Module struct {
	config           *config.Config
	pgxDb            *pgxpool.Pool
	catalogService   app.CatalogService
	workforceService app.WorkforceService
	handlers         httpPort.Handlers
}

func NewModule(
	config *config.Config,
	pgxDb *pgxpool.Pool,
	catalogService app.CatalogService,
	workforceService app.WorkforceService,
) *Module {
	if config == nil {
		panic("config can't be nil")
	}

	if pgxDb == nil {
		panic("db can't be nil")
	}

	if catalogService == nil {
		panic("catalogService can't be nil")
	}

	if workforceService == nil {
		panic("workforceService can't be nil")
	}

	return &Module{
		config:           config,
		pgxDb:            pgxDb,
		catalogService:   catalogService,
		workforceService: workforceService,
	}
}

func (m *Module) Name() string {
	return "appointments"
}

//go:embed adapters/db/migrations/*.sql
var embedMigrations embed.FS

func (m *Module) Init(ctx context.Context) error {
	if err := common.MigrateDatabaseUp(
		ctx,
		string(m.Name()),
		m.pgxDb,
		embedMigrations,
		"adapters/db/migrations",
	); err != nil {
		return err
	}

	bookingFactory := domain.NewBookingFactory(domain.BookingFactoryConfig{
		MinStartLeadTime: m.config.Appointment.MinStartLeadTime,
		MaxStartLeadTime: m.config.Appointment.MaxStartLeadTime,
		ReservationTTL:   m.config.Appointment.ReservationTTL,
	})
	appointmentRepo := db.NewAppointmentRepository(m.pgxDb)
	reservationReadModel := db.NewReservationReadModel(m.pgxDb)
	services := app.NewService(
		bookingFactory,
		appointmentRepo,
		m.catalogService,
		m.workforceService,
	)

	m.handlers = httpPort.NewHandlers(services, reservationReadModel)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, contracts *contracts.Contracts) error {
	return nil
}

func (m *Module) RegisterHttp(ctx context.Context, router common.EchoRouter) error {
	return httpPort.Register(router, m.handlers)
}
