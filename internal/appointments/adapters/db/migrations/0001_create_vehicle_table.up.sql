BEGIN;

CREATE SCHEMA IF NOT EXISTS appointments;

CREATE TABLE IF NOT EXISTS appointments.vehicles (
    vehicle_uuid    UUID PRIMARY KEY,
    user_id         VARCHAR(255) NOT NULL,
    make_code       VARCHAR(255) NOT NULL,
    model_name      VARCHAR(255) NOT NULL,
    model_year      SMALLINT NOT NULL,
    license_plate   VARCHAR(255)
);

COMMIT;
