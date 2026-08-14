BEGIN;

CREATE INDEX IF NOT EXISTS idx_appointment_dealer_ship_id_window
    ON appointments.appointment (dealer_ship_id, start_time, estimated_end_time);

CREATE INDEX IF NOT EXISTS idx_reservation_appointment_uuid
    ON appointments.reservation (appointment_uuid);

COMMIT;
