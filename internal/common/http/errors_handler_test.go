package http_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"scheduler/internal/common"
	commonHttp "scheduler/internal/common/http"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"github.com/labstack/echo/v4"
)

func TestEchoErrorHandler(t *testing.T) {
	testCases := []struct {
		Name               string
		Error              error
		ExpectedResponse   commonHttp.HttpErrorResponse
		ExpectedStatusCode int
	}{
		{
			Name:  "generic_error_returns_500",
			Error: errors.New("something went wrong"),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "something went wrong, please try again later",
				Slug:    "internal_server_error",
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		{
			Name:  "echo_http_error_with_404",
			Error: echo.NewHTTPError(http.StatusNotFound),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Not Found",
				Slug:    "not_found",
			},
			ExpectedStatusCode: http.StatusNotFound,
		},
		{
			Name:  "echo_http_error_with_400",
			Error: echo.NewHTTPError(http.StatusBadRequest),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Bad Request",
				Slug:    "bad_request",
			},
			ExpectedStatusCode: http.StatusBadRequest,
		},
		{
			Name: "common_error_with_all_fields",
			Error: common.Error{
				HttpErrorCode: http.StatusUnprocessableEntity,
				PublicError:   "Invalid order data",
				ErrorSlug:     "invalid_order",
				InternalError: errors.New("internal details"),
				Details: []common.ErrorDetails{
					{
						EntityType: "order",
						EntityID:   "456",
						ErrorSlug:  "missing_customer_id",
						Message:    "customer_id is required",
					},
				},
			},
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Invalid order data",
				Slug:    "invalid_order",
				Details: []commonHttp.HttpErrorDetail{
					{
						EntityType: "order",
						EntityID:   "456",
						ErrorSlug:  "missing_customer_id",
						Message:    "customer_id is required",
					},
				},
			},
			ExpectedStatusCode: http.StatusUnprocessableEntity,
		},
		{
			Name: "common_error_with_only_public_error",
			Error: common.Error{
				PublicError: "User not found",
			},
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "User not found",
				Slug:    "internal_server_error",
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		{
			Name: "common_error_overrides_status_code",
			Error: common.Error{
				HttpErrorCode: http.StatusConflict,
				PublicError:   "Resource already exists",
				ErrorSlug:     "resource_exists",
			},
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Resource already exists",
				Slug:    "resource_exists",
			},
			ExpectedStatusCode: http.StatusConflict,
		},
		{
			Name:  "new_not_found_error_helper",
			Error: common.NewNotFoundError("order_not_found", "Order not found"),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Order not found",
				Slug:    "order_not_found",
			},
			ExpectedStatusCode: http.StatusNotFound,
		},
		{
			Name:  "new_invalid_input_error_helper",
			Error: common.NewInvalidInputError("invalid_email", "Email format is invalid"),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Email format is invalid",
				Slug:    "invalid_email",
			},
			ExpectedStatusCode: http.StatusBadRequest,
		},
		{
			Name: "common_error_with_details_and_internal_error",
			Error: common.Error{
				HttpErrorCode: http.StatusBadRequest,
				PublicError:   "Validation failed",
				ErrorSlug:     "validation_error",
				InternalError: errors.New("database constraint violated"),
				Details: []common.ErrorDetails{
					{
						EntityType: "email",
						EntityID:   "456",
						ErrorSlug:  "invalid_email",
						Message:    "email format is incorrect",
					},
					{
						EntityType: "name",
						EntityID:   "456",
						ErrorSlug:  "missing_name",
						Message:    "name is required",
					},
				},
			},
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Validation failed",
				Slug:    "validation_error",
				Details: []commonHttp.HttpErrorDetail{
					{
						EntityType: "email",
						EntityID:   "456",
						ErrorSlug:  "invalid_email",
						Message:    "email format is incorrect",
					},
					{
						EntityType: "name",
						EntityID:   "456",
						ErrorSlug:  "missing_name",
						Message:    "name is required",
					},
				},
			},
			ExpectedStatusCode: http.StatusBadRequest,
		},
		{
			Name: "common_error_uses_default_status_when_not_specified",
			Error: common.Error{
				PublicError: "Something went wrong",
				ErrorSlug:   "custom_error",
			},
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Something went wrong",
				Slug:    "custom_error",
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		{
			Name: "with_internal_error_does_not_expose_internal_details",
			Error: common.Error{
				HttpErrorCode: http.StatusNotFound,
				PublicError:   "Order not found",
				ErrorSlug:     "order_not_found",
			}.WithInternalError(errors.New("database connection failed: timeout")),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Order not found",
				Slug:    "order_not_found",
			},
			ExpectedStatusCode: http.StatusNotFound,
		},
		{
			Name: "with_details_adds_details_to_response",
			Error: common.Error{
				HttpErrorCode: http.StatusBadRequest,
				PublicError:   "Invalid request data",
				ErrorSlug:     "invalid_input",
			}.WithDetails([]common.ErrorDetails{
				{
					EntityType: "email",
					EntityID:   "456",
					ErrorSlug:  "invalid_email",
					Message:    "email format is incorrect",
				},
				{
					EntityType: "name",
					EntityID:   "456",
					ErrorSlug:  "missing_name",
					Message:    "name is required",
				},
			}),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Invalid request data",
				Slug:    "invalid_input",
				Details: []commonHttp.HttpErrorDetail{
					{
						EntityType: "email",
						EntityID:   "456",
						ErrorSlug:  "invalid_email",
						Message:    "email format is incorrect",
					},
					{
						EntityType: "name",
						EntityID:   "456",
						ErrorSlug:  "missing_name",
						Message:    "name is required",
					},
				},
			},
			ExpectedStatusCode: http.StatusBadRequest,
		},
		{
			Name: "with_details_appends_to_existing",
			Error: common.Error{
				HttpErrorCode: http.StatusBadRequest,
				PublicError:   "Invalid request data",
				ErrorSlug:     "invalid_input",
			}.WithDetails([]common.ErrorDetails{
				{
					EntityType: "email",
					EntityID:   "456",
					ErrorSlug:  "invalid_email",
					Message:    "email format is incorrect",
				},
			}).WithDetails([]common.ErrorDetails{
				{
					EntityType: "name",
					EntityID:   "456",
					ErrorSlug:  "missing_name",
					Message:    "name is required",
				},
			}),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Invalid request data",
				Slug:    "invalid_input",
				Details: []commonHttp.HttpErrorDetail{
					{
						EntityType: "email",
						EntityID:   "456",
						ErrorSlug:  "invalid_email",
						Message:    "email format is incorrect",
					},
					{
						EntityType: "name",
						EntityID:   "456",
						ErrorSlug:  "missing_name",
						Message:    "name is required",
					},
				},
			},
			ExpectedStatusCode: http.StatusBadRequest,
		},
		{
			Name: "with_internal_error_and_with_details_chained",
			Error: common.Error{
				HttpErrorCode: http.StatusNotFound,
				PublicError:   "Customer not found",
				ErrorSlug:     "customer_not_found",
			}.WithInternalError(errors.New("SELECT failed: connection timeout")).
				WithDetails([]common.ErrorDetails{
					{
						EntityType: "email",
						EntityID:   "456",
						ErrorSlug:  "invalid_email",
						Message:    "email format is incorrect",
					},
					{
						EntityType: "name",
						EntityID:   "456",
						ErrorSlug:  "missing_name",
						Message:    "name is required",
					},
				}),
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Customer not found",
				Slug:    "customer_not_found",
				Details: []commonHttp.HttpErrorDetail{
					{
						EntityType: "email",
						EntityID:   "456",
						ErrorSlug:  "invalid_email",
						Message:    "email format is incorrect",
					},
					{
						EntityType: "name",
						EntityID:   "456",
						ErrorSlug:  "missing_name",
						Message:    "name is required",
					},
				},
			},
			ExpectedStatusCode: http.StatusNotFound,
		},
		{
			Name: "internal_error_with_sensitive_data_not_exposed",
			Error: common.Error{
				HttpErrorCode: http.StatusInternalServerError,
				PublicError:   "Database operation failed",
				ErrorSlug:     "database_error",
				InternalError: errors.New("failed to connect to postgres://admin:secretpass@db:5432/prod"),
			},
			ExpectedResponse: commonHttp.HttpErrorResponse{
				Message: "Database operation failed",
				Slug:    "database_error",
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			commonHttp.EchoErrorHandler(tc.Error, c)

			assert.Equal(t, tc.ExpectedStatusCode, rec.Code)

			var response commonHttp.HttpErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err, "should be valid JSON")

			AssertJsonRepresentationEqual(t, tc.ExpectedResponse, response)
		})
	}
}

func AssertJsonRepresentationEqual(t *testing.T, expected, actual any) {
	t.Helper()

	expectedJson, err := json.Marshal(expected)
	require.NoError(t, err, "failed to marshal expected value to JSON")

	actualJson, err := json.Marshal(actual)
	require.NoError(t, err, "failed to marshal actual value to JSON")

	assert.JSONEq(t, string(expectedJson), string(actualJson), "JSON representations should be equal")
}
