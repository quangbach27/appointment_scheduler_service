package domain

import (
	"time"

	"scheduler/internal/common"
)

type ReservationUUID struct {
	common.UUID
}

type Reservation struct {
	uuid            ReservationUUID
	appointmentUUID AppointmentUUID

	createdAt time.Time
	expiredAt time.Time
}

func (r *Reservation) UUID() ReservationUUID {
	return r.uuid
}

func (r *Reservation) AppointmentUUID() AppointmentUUID {
	return r.appointmentUUID
}

func (r *Reservation) CreatedAt() time.Time {
	return r.createdAt
}

func (r *Reservation) ExpiredAt() time.Time {
	return r.expiredAt
}

func (r *Reservation) IsExpired() bool {
	return !time.Now().Before(r.expiredAt)
}
