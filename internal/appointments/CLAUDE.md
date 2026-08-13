# internal/appointments/CLAUDE.md

Specifics for the `appointments` context; the root `CLAUDE.md` covers repo-wide commands, infra, and the generic layout. Domain rules live only here.

## What a booking is

Booking is **two-phase**: requesting one allocates a technician and a bay and writes a `pending` appointment, guarded by a short-lived `Reservation` hold that the caller must confirm before it expires.

`pending ──confirm (hold still valid)──> confirmed ──cancel──> cancelled`, and a hold left to expire frees the slot with nothing written.

Two objects, deliberately separate, both minted by one `BookingFactory`:

- **`Appointment`** — the booking: dealer, service bay, technician, user, service type, vehicle, start/end time, plus a `pending` / `confirmed` / `cancelled` status.
- **`Reservation`** — the hold: its UUID, the appointment it guards, `expiredAt`, and the caller's `idempotencyKey`.

## Domain rules

- **Start time** must fall in the bookable window: `MinStartLeadTime` (1h) to `MaxStartLeadTime` (30 days, `APPOINTMENT_MAX_START_LEAD_TIME_DAYS`). One definition, also exported as `ValidateStartTime` so `app` can reject early.
- **`idempotencyKey`** is required and `UNIQUE` in the DB, and is checked three times, cheapest first: `RequestBooking` calls `FindReservationByIdempotencyKey` before any external call so a retry skips catalog/workforce and the booking transaction entirely; `RequestAppointment` re-checks the same key inside the SERIALIZABLE tx before touching availability; `IsUniqueViolationError` after the tx backstops two identical requests that raced past both checks. All three return the *original* reservation on replay.
- **`Confirm`** — `pending` → `confirmed`, rejecting in order: `reservation-mismatch`, `appointment-not-pending`, `reservation-expired`. A failed confirm never mutates status.
- **`Cancel`** — `confirmed` → `cancelled` only: `appointment-already-cancelled` (409), `appointment-not-confirmed` (409), `appointment-in-the-past` (400).
- **Availability** is decided from our own rows, never the workforce service: a resource is busy if an overlapping appointment is `confirmed`, or `pending` with an unexpired hold. Overlap is half-open, so back-to-back bookings are fine. It's computed once per booking attempt — `GetBusyServiceBaysAndTechnicians` returns every busy `(service_bay_id, technician_id)` pair for the dealer in the requested window, not a check of one candidate pair at a time.
- **Two expiries**, easy to confuse. The *hold* expiry (`Reservation.expiredAt`, 5m TTL) closes the checkout window and frees the slot with no row mutation. The *no-show* expiry (`Appointment.IsExpired()`: pending with a past start) is computed on read and never stored — there is no `expired` status.

## Ports

`domain.AppointmentRepository` (implemented in `adapters/db`):

- `FindReservationByIdempotencyKey` — a plain, non-transactional read; the fast-path idempotency check described above. Not the source of truth, just an optimization.
- `RequestAppointment` runs at SERIALIZABLE, takes a `RequestAppointmentParams` (user, idempotency key, dealer, `[start, end)` window, vehicle) plus a `createFn` callback, and returns the *effective* `ReservationUUID` — on an idempotency replay that is the stored original's, so use the return value, not the reservation `createFn` built. Inside the tx the repository resolves idempotency, runs `GetBusyServiceBaysAndTechnicians`, and calls `createFn(busyServiceBayIDs, busyTechnicianIDs)` — two `map[string]string` lookups (bay ID → occupying technician ID, and the reverse) built from those rows. `createFn` is supplied by `app`: it picks a technician/bay from the candidates it already fetched from `WorkforceService` and mints the `Appointment`/`Reservation` via `BookingFactory`, or returns `no-availability` if nothing in its candidate lists is free. Because the busy set is complete before `createFn` runs, there's no retry-a-different-candidate loop — one call either succeeds or the whole booking fails.
- `ConfirmAppointment` / `CancelAppointment` run at RepeatableRead.

`app` declares two outbound service ports, stubbed in `adapters/catalog` and `adapters/workforce`:

- **`CatalogService`** — service type → duration + required certifications.
- **`WorkforceService`** — a dealership's technicians (each with a `ReviewScore`, used to prefer the best-reviewed one still available) and bays, filtered to those **on duty** for the requested window (not off that day, not under maintenance); it knows nothing about who is already booked. The stub models this with per-resource blackout windows, plus `dealer-offday` / `dealer-maintenance` permanently blacked out for tests.

## `RequestBooking`

Returns just a `ReservationUUID`: `ValidateStartTime` → idempotency fast path (`FindReservationByIdempotencyKey`, returns early on a hit) → catalog lookup → workforce candidates for `[start, start+duration)` → sort technicians by `ReviewScore` descending → `NewVehicle` → one call to `appointmentRepo.RequestAppointment` with a `createFn` closure over the candidate lists. `createFn` walks the sorted technicians and the bays for the first ID absent from the busy maps the repository built inside the tx; if either search comes up empty it returns `no-availability` (409). Every external call happens before the transaction, which is retried on serialization failure and would otherwise re-issue them.

## HTTP

`openapi.yaml` is the source of truth, including each endpoint's error slugs.

| Endpoint | Success |
|---|---|
| `POST /appointments` (`requestBooking`) | `201 {reservationUuid}` |
| `POST /reservations/{uuid}/confirm` | `200 {appointmentUuid}` |
| `GET /reservations/{uuid}` | `200` reservation + appointment |
| `POST /appointments/{uuid}/cancel` | `204` |

- `requestBooking` is a command: it returns an identifier only, and clients follow up with `getReservation` for the details and hold expiry.
- `X-User-Id` and `Idempotency-Key` are **optional in the spec** deliberately — with `required: true` the generated binder returns a generic `400` before the handler runs, making the documented `401 missing-user-id` unreachable. Handlers check them instead. There is no auth middleware yet.
- `getReservation` is served by the `ReservationReadModel` port: a CQRS read side over one joined sqlc query, returning the OpenAPI DTO directly.

## Wiring and testing

`module.go` runs the embedded migrations, then builds factory → repository → read model → `app.Service` → `Handlers`; `internal/svc.go` mounts it, with the catalog and workforce stubs arriving via `internal.ExternalService`. By convention `app/` and `ports/http/` carry no unit tests — component tests cover them end to end. `adapters/workforce`'s stub gives `dealer-1`'s technicians distinct `ReviewScore`s so a component test can assert the highest-scored available one wins the booking.
