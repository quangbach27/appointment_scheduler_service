-- name: GetAppointmentByUUID :one
SELECT * FROM appointments.appointment
WHERE appointment_uuid = $1;

-- name: CreateAppointment :exec
INSERT INTO appointments.appointment (appointment_uuid, dealer_ship_id, service_bay_id, technician_id, user_id, service_type, vehicle_uuid, start_time, estimated_end_time, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetAppointmentByUUIDAndUserID :one
SELECT * FROM appointments.appointment
WHERE appointment_uuid = $1 AND user_id = $2;

-- name: UpdateAppointmentStatus :exec
UPDATE appointments.appointment
SET status = $2
WHERE appointment_uuid = $1;
