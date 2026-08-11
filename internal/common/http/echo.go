package http

import (
	"log/slog"
	"net/http"
	"time"

	"scheduler/internal/common"
	"scheduler/internal/common/config"

	"github.com/labstack/echo/v4"
)

func NewEcho(appConfig *config.AppConfig) *echo.Echo {
	if appConfig == nil {
		panic("appConfig can't be nil")
	}

	e := echo.New()
	e.HideBanner = true

	useMiddlewares(e, appConfig)

	e.HTTPErrorHandler = EchoErrorHandler
	e.Logger = common.NewEchoSlogAdapter(slog.Default())

	if appConfig.Env != "dev" {
		e.Server.WriteTimeout = 30 * time.Second
		e.Server.ReadHeaderTimeout = 30 * time.Second
		e.Server.ReadTimeout = 30 * time.Second
		e.Server.IdleTimeout = 60 * time.Second
	}

	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	return e
}
