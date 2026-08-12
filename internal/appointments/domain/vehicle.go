package domain

import (
	"fmt"
	"strings"
	"time"

	"scheduler/internal/common"
)

type VehicleUUID struct {
	common.UUID
}

type Vehicle struct {
	uuid         VehicleUUID
	makeCode     string
	modelName    string
	modelYear    int
	licensePlate string
}

func NewVehicle(
	makeCode string,
	modelName string,
	modelYear int,
	licensePlate string,
) (*Vehicle, error) {
	modelName = strings.TrimSpace(modelName)
	makeCode = strings.TrimSpace(makeCode)

	errDetails := []common.ErrorDetails{}
	if modelName == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Vehicle",
			EntityID:   "modelName",
			ErrorSlug:  "required",
			Message:    "modelName is required",
		})
	}

	if makeCode == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Vehicle",
			EntityID:   "makeCode",
			ErrorSlug:  "required",
			Message:    "makeCode is required",
		})
	}

	minYear := 1886
	maxYear := time.Now().Year() + 1
	if modelYear < minYear || modelYear > maxYear {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Vehicle",
			EntityID:   "year",
			ErrorSlug:  "out-of-range",
			Message:    fmt.Sprintf("year must be between %d and %d", minYear, maxYear),
		})
	}

	if len(errDetails) > 0 {
		return nil, common.NewInvalidInputError("invalid-vehicle", "invalid vehicle input").WithDetails(errDetails)
	}

	return &Vehicle{
		uuid:         VehicleUUID{common.NewUUIDv7()},
		modelName:    modelName,
		makeCode:     makeCode,
		modelYear:    modelYear,
		licensePlate: licensePlate,
	}, nil
}

func (v *Vehicle) UUID() VehicleUUID {
	return v.uuid
}

func (v *Vehicle) ModelName() string {
	return v.modelName
}

func (v *Vehicle) MakeCode() string {
	return v.makeCode
}

func (v *Vehicle) ModelYear() int {
	return v.modelYear
}

func (v *Vehicle) LicensePlate() string {
	return v.licensePlate
}
