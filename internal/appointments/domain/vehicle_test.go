package domain_test

import (
	"testing"
	"time"

	"scheduler/internal/appointments/domain"
	"scheduler/internal/common"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestNewVehicle_Validation(t *testing.T) {
	t.Parallel()

	currentYear := time.Now().Year()
	tests := []struct {
		name          string
		makeCode      string
		modelName     string
		modelYear     int
		licensePlate  string
		expectErr     bool
		expectedSlugs []string
	}{
		{
			name:         "returns vehicle when input is valid",
			makeCode:     "HONDA",
			modelName:    "Civic",
			modelYear:    currentYear,
			licensePlate: "51A-12345",
			expectErr:    false,
		},
		{
			name:          "returns required when makeCode is empty",
			makeCode:      "",
			modelName:     "Civic",
			modelYear:     currentYear,
			licensePlate:  "51A-12345",
			expectErr:     true,
			expectedSlugs: []string{"required"},
		},
		{
			name:          "returns required when modelName is empty",
			makeCode:      "HONDA",
			modelName:     "",
			modelYear:     currentYear,
			licensePlate:  "51A-12345",
			expectErr:     true,
			expectedSlugs: []string{"required"},
		},
		{
			name:          "returns out of range when year is below minimum",
			modelName:     "Civic",
			makeCode:      "HONDA",
			modelYear:     1885,
			licensePlate:  "51A-12345",
			expectErr:     true,
			expectedSlugs: []string{"out-of-range"},
		},
		{
			name:          "returns out of range when year is above maximum",
			modelName:     "Civic",
			makeCode:      "HONDA",
			modelYear:     currentYear + 2,
			licensePlate:  "51A-12345",
			expectErr:     true,
			expectedSlugs: []string{"out-of-range"},
		},
		{
			name:         "accepts year lower boundary",
			modelName:    "Model T",
			makeCode:     "FORD",
			modelYear:    1886,
			licensePlate: "01A-00001",
			expectErr:    false,
		},
		{
			name:         "accepts year upper boundary",
			modelName:    "Future",
			makeCode:     "TEST",
			modelYear:    currentYear + 1,
			licensePlate: "99Z-99999",
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vehicle, err := domain.NewVehicle(tt.makeCode, tt.modelName, tt.modelYear, tt.licensePlate)

			if tt.expectErr {
				require.Error(t, err)

				var invalidErr common.Error
				require.ErrorAs(t, err, &invalidErr)

				assert.Equal(t, "invalid-vehicle", invalidErr.ErrorSlug)
				assert.Equal(t, "invalid vehicle input", invalidErr.PublicError)

				require.Len(t, invalidErr.Details, len(tt.expectedSlugs))
				for i, expectedSlug := range tt.expectedSlugs {
					assert.Equal(t, expectedSlug, invalidErr.Details[i].ErrorSlug)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.modelName, vehicle.ModelName())
			assert.Equal(t, tt.makeCode, vehicle.MakeCode())
			assert.Equal(t, tt.modelYear, vehicle.ModelYear())
			assert.Equal(t, tt.licensePlate, vehicle.LicensePlate())
		})
	}
}
