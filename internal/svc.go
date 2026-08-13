package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	"scheduler/internal/appointments"
	appointmentsApp "scheduler/internal/appointments/app"
	"scheduler/internal/common/config"
	commonHttp "scheduler/internal/common/http"
	"scheduler/internal/common/log"
	"scheduler/internal/common/module"
	"scheduler/internal/common/module/contracts"
)

type ExternalService struct {
	AppointmentCatalog   appointmentsApp.CatalogService
	AppointmentWorkforce appointmentsApp.WorkforceService
}

type Svc struct {
	config     *config.Config
	echoRouter *echo.Echo
	pgxDb      *pgxpool.Pool

	modules []module.Module
}

func New(
	ctx context.Context,
	config *config.Config,
	pgxDb *pgxpool.Pool,
	externalService ExternalService,
) (Svc, error) {
	if config == nil {
		return Svc{}, errors.New("config can't be nil")
	}

	router := commonHttp.NewEcho(config.App)

	moduleContracts := &contracts.Contracts{}

	modules := []module.Module{
		appointments.NewModule(
			config,
			pgxDb,
			externalService.AppointmentCatalog,
			externalService.AppointmentWorkforce,
		),
	}

	for _, module := range modules {
		start := time.Now()

		if err := module.Init(ctx); err != nil {
			return Svc{}, fmt.Errorf("initializing module %s failed: %w", module.Name(), err)
		}

		if err := module.RegisterContracts(ctx, moduleContracts); err != nil {
			return Svc{}, fmt.Errorf("failed to register contracts: %w", err)
		}

		log.FromContext(ctx).With(
			"duration", time.Since(start),
			"module", module.Name(),
		).Debug("Initialized module")
	}

	if err := moduleContracts.Verify(); err != nil {
		return Svc{}, fmt.Errorf("verifying module contracts failed: %w", err)
	}

	for _, module := range modules {
		err := module.RegisterHttp(ctx, router)
		if err != nil {
			return Svc{}, fmt.Errorf("registering http for module %s failed: %w", module.Name(), err)
		}
	}

	return Svc{
		config:     config,
		echoRouter: router,
		pgxDb:      pgxDb,
		modules:    modules,
	}, nil
}

func (s Svc) Run(ctx context.Context, port string) error {
	g, ctx := errgroup.WithContext(ctx)

	// Start http server
	g.Go(func() error {
		if err := s.echoRouter.Start(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("starting http server failed: %w", err)
		}
		return nil
	})

	// This goroutine handle graceful shutdown
	g.Go(func() error {
		<-ctx.Done()
		defer s.pgxDb.Close()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.echoRouter.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutting down http server failed: %w", err)
		}

		return nil
	})

	return g.Wait()
}
