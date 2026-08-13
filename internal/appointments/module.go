package appointments

import (
	"context"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"

	httpPort "scheduler/internal/appointments/ports/http"
	"scheduler/internal/common"
	"scheduler/internal/common/config"
	"scheduler/internal/common/module/contracts"
)

type Module struct {
	config   *config.Config
	pgxDb    *pgxpool.Pool
	handlers httpPort.Handlers
}

func NewModule(
	config *config.Config,
	pgxDb *pgxpool.Pool,
) *Module {
	if pgxDb == nil {
		panic("db can't be nil")
	}

	return &Module{
		config: config,
		pgxDb:  pgxDb,
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

	m.handlers = httpPort.NewHandlers(nil)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, contracts *contracts.Contracts) error {
	return nil
}

func (m *Module) RegisterHttp(
	ctx context.Context,
	router common.EchoRouter,
) error {
	return httpPort.Register(router, m.handlers)
}
