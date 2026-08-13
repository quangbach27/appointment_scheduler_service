-- name: GetReservationByIdempotencyKey :one
SELECT * FROM appointments.reservation
WHERE idempotency_key = $1;

-- name: CreateReservation :exec
INSERT INTO appointments.reservation (reservation_uuid, appointment_uuid, created_at, expired_at, idempotency_key)
VALUES ($1, $2, $3, $4, $5);

-- name: GetReservationByUUID :one
SELECT * FROM appointments.reservation
WHERE reservation_uuid = $1;

-- name: GetReservationWithAppointmentByUUID :one
SELECT
    sqlc.embed(r),
    sqlc.embed(a)
FROM appointments.reservation r
JOIN appointments.appointment a ON a.appointment_uuid = r.appointment_uuid
WHERE r.reservation_uuid = $1 AND a.user_id = $2;
