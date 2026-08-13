-- name: GetReservationByIdempotencyKey :one
SELECT * FROM appointments.reservation
WHERE idempotency_key = $1;

-- name: CreateReservation :one
INSERT INTO appointments.reservation (reservation_uuid, appointment_uuid, created_at, expired_at, idempotency_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetReservationByUUID :one
SELECT * FROM appointments.reservation
WHERE reservation_uuid = $1;

-- name: GetReservationWithAppointmentByUUID :one
SELECT
    r.reservation_uuid,
    r.expired_at,
    a.appointment_uuid,
    a.status,
    a.dealer_ship_id,
    a.service_bay_id,
    a.technician_id,
    a.service_type,
    a.vehicle_uuid,
    a.start_time,
    a.estimated_end_time
FROM appointments.reservation r
JOIN appointments.appointment a ON a.appointment_uuid = r.appointment_uuid
WHERE r.reservation_uuid = $1 AND a.user_id = $2;
