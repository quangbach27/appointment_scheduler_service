package testutils

import (
	"context"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler/internal/common"
	"scheduler/internal/common/config"
)

func RunMigrations(moduleName string, embedFS fs.FS, migrationsDir string) {
	ctx := context.Background()
	dbCfg := config.NewConfig().DB

	pool, err := pgxpool.New(ctx, dbCfg.URL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	if err := common.MigrateDatabaseUp(ctx, moduleName, pool, embedFS, migrationsDir); err != nil {
		panic(err)
	}
}

func NewDB() *pgxpool.Pool {
	dbCfg := config.NewConfig().DB

	config, err := pgxpool.ParseConfig(dbCfg.URL)
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}

	return pool
}
