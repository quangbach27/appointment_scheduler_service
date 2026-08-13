package module

import (
	"context"

	"scheduler/internal/common"
	"scheduler/internal/common/module/contracts"
)

type Module interface {
	Name() string
	Init(ctx context.Context) error
	RegisterHttp(
		ctx context.Context,
		router common.EchoRouter,
	) error
	RegisterContracts(ctx context.Context, contracts *contracts.Contracts) error
}
