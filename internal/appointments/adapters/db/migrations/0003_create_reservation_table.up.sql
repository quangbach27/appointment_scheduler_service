BEGIN;

CREATE TABLE IF NOT EXISTS appointments.reservation (
    reservation_uuid  UUID PRIMARY KEY,
    appointment_uuid  UUID NOT NULL REFERENCES appointments.appointment (appointment_uuid),
    created_at        TIMESTAMPTZ NOT NULL,
    expired_at        TIMESTAMPTZ NOT NULL,
    idempotency_key   VARCHAR(255) NOT NULL,

    CONSTRAINT uq_reservation_idempotency_key UNIQUE (idempotency_key)
);

COMMIT;
