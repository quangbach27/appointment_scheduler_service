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
