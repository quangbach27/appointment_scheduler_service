package domain

import "scheduler/internal/common"

type AppointmentStatus struct {
	common.Enum[AppointmentStatusValues]
}

type AppointmentStatusValues string

func (v AppointmentStatusValues) Values() []string {
	return []string{"pending", "confirmed", "cancelled"}
}

var (
	AppointmentStatusPending   = common.MustEnum[AppointmentStatus]("pending")
	AppointmentStatusConfirmed = common.MustEnum[AppointmentStatus]("confirmed")
	AppointmentStatusCanceled  = common.MustEnum[AppointmentStatus]("cancelled")
)
