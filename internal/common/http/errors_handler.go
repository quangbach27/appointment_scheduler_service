package http

import (
	"errors"
	"net/http"
	"strings"

	"scheduler/internal/common"
	"scheduler/internal/common/log"

	"github.com/labstack/echo/v4"
)

func EchoErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	ctx := c.Request().Context()
	logger := log.FromContext(ctx)

	httpErrorResponse, httpStatus := httpErrorResponseFromErr(err)

	logger.Error("Handling HTTP error", "error", err)

	if err := c.JSON(httpStatus, httpErrorResponse); err != nil {
		logger.Error("Failed to send error response", "error", err)
	}
}

type HttpErrorResponse struct {
	Message string            `json:"message"`
	Slug    string            `json:"slug"`
	Details []HttpErrorDetail `json:"details,omitempty"`
}

type HttpErrorDetail struct {
	EntityType string `json:"entity_type,omitempty"`
	EntityID   string `json:"entity_id,omitempty"`
	ErrorSlug  string `json:"error_slug,omitempty"`
	Message    string `json:"message,omitempty"`
}

func httpErrorResponseFromErr(err error) (HttpErrorResponse, int) {
	publicError := "something went wrong, please try again later"
	statusCode := http.StatusInternalServerError
	errorSlug := "internal_server_error"

	var he *echo.HTTPError
	if errors.As(err, &he) {
		statusCode = he.Code
		publicError = http.StatusText(statusCode)
		errorSlug = strings.ToLower(strings.ReplaceAll(publicError, " ", "_"))

		return HttpErrorResponse{
			Slug:    errorSlug,
			Message: publicError,
		}, statusCode
	}

	var commonErr common.Error
	if errors.As(err, &commonErr) {
		if commonErr.PublicError != "" {
			publicError = commonErr.PublicError
		}
		if commonErr.ErrorSlug != "" {
			errorSlug = commonErr.ErrorSlug
		}
		if commonErr.HttpErrorCode != 0 {
			statusCode = commonErr.HttpErrorCode
		}
	}

	httpDetails := make([]HttpErrorDetail, 0, len(commonErr.Details))
	for _, detail := range commonErr.Details {
		httpDetails = append(httpDetails, HttpErrorDetail(detail))
	}

	return HttpErrorResponse{
		Slug:    errorSlug,
		Message: publicError,
		Details: httpDetails,
	}, statusCode
}
