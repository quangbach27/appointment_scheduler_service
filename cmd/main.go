package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler/internal"
	"scheduler/internal/appointments/adapters/catalog"
	"scheduler/internal/appointments/adapters/workforce"
	"scheduler/internal/common/config"
	"scheduler/internal/common/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Init(slog.LevelInfo)

	config := config.NewConfig()

	dbPgx, err := pgxpool.New(ctx, config.DB.URL)
	if err != nil {
		panic(err)
	}

	externalService := internal.ExternalService{
		AppointmentCatalog:   catalog.NewStubCatalogService(),
		AppointmentWorkforce: workforce.NewStubWorkforceService(),
	}
	svc, err := internal.New(
		ctx,
		config,
		dbPgx,
		externalService,
	)
	if err != nil {
		panic(err)
	}

	if err := svc.Run(ctx, fmt.Sprintf(":%s", config.App.Port)); err != nil {
		panic(err)
	}
}
