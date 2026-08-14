-- name: GetAppointmentByUUID :one
SELECT * FROM appointments.appointment
WHERE appointment_uuid = $1;

-- name: CreateAppointment :exec
INSERT INTO appointments.appointment (appointment_uuid, dealer_ship_id, service_bay_id, technician_id, user_id, service_type, vehicle_uuid, start_time, estimated_end_time, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetAppointmentByUUIDAndUserID :one
SELECT * FROM appointments.appointment
WHERE user_id = $1 AND appointment_uuid = $2;

-- name: UpdateAppointmentStatus :exec
UPDATE appointments.appointment
SET status = $2
WHERE appointment_uuid = $1;

-- Returns the (service_bay_id, technician_id) pairs busy for the requested
-- window: busy if an overlapping appointment is confirmed, or pending with an
-- unexpired hold. Overlap is half-open, so back-to-back appointments don't
-- conflict.
-- name: GetBusyServiceBaysAndTechnicians :many
SELECT DISTINCT
    a.service_bay_id,
    a.technician_id
FROM appointments.appointment a
LEFT JOIN appointments.reservation r
    ON r.appointment_uuid = a.appointment_uuid
WHERE
    a.dealer_ship_id = @dealer_ship_id
    AND a.start_time < @end_time
    AND a.estimated_end_time > @start_time
    AND (
        a.status = 'confirmed'
        OR (a.status = 'pending' AND r.expired_at > now())
    );

