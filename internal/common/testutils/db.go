package testutils

import (
	"context"
	"io/fs"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler/internal/common"
	"scheduler/internal/common/config"
)

var (
	dbOnce sync.Once
	dbPool *pgxpool.Pool
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
	dbOnce.Do(func() {
		dbCfg := config.NewConfig().DB

		poolConfig, err := pgxpool.ParseConfig(dbCfg.URL)
		if err != nil {
			panic(err)
		}

		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			panic(err)
		}

		dbPool = pool
	})

	return dbPool
}
