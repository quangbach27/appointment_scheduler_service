-- name: UpsertVehicle :exec
INSERT INTO appointments.vehicles (vehicle_uuid, user_id, make_code, model_name, model_year, license_plate)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (vehicle_uuid) DO NOTHING;

-- name: GetVehicleByUUID :one
SELECT * FROM appointments.vehicles
WHERE vehicle_uuid = $1;
