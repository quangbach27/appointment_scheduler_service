BEGIN;

CREATE TYPE appointments.appointment_status AS ENUM ('pending', 'confirmed', 'cancelled');

CREATE TABLE IF NOT EXISTS appointments.appointment (
    appointment_uuid     UUID PRIMARY KEY,
    dealer_ship_id       VARCHAR(255) NOT NULL,
    service_bay_id       VARCHAR(255) NOT NULL,
    technician_id        VARCHAR(255) NOT NULL,
    user_id              VARCHAR(255) NOT NULL,
    service_type         VARCHAR(255) NOT NULL,
    vehicle_uuid         UUID NOT NULL REFERENCES appointments.vehicles (vehicle_uuid),
    start_time           TIMESTAMPTZ NOT NULL,
    estimated_end_time   TIMESTAMPTZ NOT NULL,
    status               appointments.appointment_status NOT NULL,

    CONSTRAINT chk_appointment_time_range CHECK (estimated_end_time > start_time)
);

COMMIT;
