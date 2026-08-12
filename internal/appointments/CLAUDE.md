# internal/appointments/CLAUDE.md

This supplements the root `CLAUDE.md` (which covers repo-wide commands, infra, and the generic bounded-context layout pattern) with specifics for the `appointments` bounded context.

## Current wiring state

- `module.go`'s `Init` calls `httpPort.NewHandlers(nil)` instead of constructing a real `app.Service` — the module isn't fully wired yet.
- `appointments.NewModule` is not in `internal/svc.go`'s module slice, so this context isn't actually mounted into the running server yet.
- `ports/http.Register(router, handlers)` is currently a no-op — no HTTP routes are registered.
- `app.Service` (`app/service.go`) is built from `*domain.AppointmentFactory`, `*domain.ReservationFactory`, `domain.AppointmentRepository`, and a `CatalogService` port it defines itself (`GetServiceEntry`).
- The three use cases — `RequestBooking` (`request_appointment.go`), `ConfirmAppointment` (`confirm_appointment.go`), `CancelAppointment` (`cancel_appointment.go`) — are **all still no-op stubs**: empty command structs, zero-value returns, no business logic, no repository calls.
- `domain.AppointmentRepository` (the outbound persistence port) has no implementation yet — nothing under `adapters/db/` implements it.

## Appointment domain model

A booking has two related-but-distinct objects, deliberately kept separate:

- **`Appointment`** (`domain/appointment.go`) owns the actual booking details (`AppointmentDetails`: dealer, service bay, technician, user, service type, vehicle, start/end time) and a 3-value status enum — `pending` / `confirmed` / `cancelled` (`domain/appointment_status.go`). It's created directly in `Pending` via `AppointmentFactory.CreateAppointment` (`domain/factory.go`), which validates required fields and enforces `MinStartLeadTime`.
- **`Reservation`** (`domain/reservation.go`) is a short-lived, detail-free token: just a `ReservationUUID`, the `AppointmentUUID` it guards, and a TTL-based `expiredAt`. Minted via `ReservationFactory.CreateReservation` (`domain/factory.go`), which does no validation — it only stamps `ReservationTTL` (default 5m, `defaultReservationTTL`).
- **`Appointment.Confirm(reservation *Reservation)`** transitions `Pending` → `Confirmed`. It checks the reservation actually belongs to that appointment (`reservation-mismatch`), that the appointment is still `Pending` (`appointment-not-pending`), and that the reservation hasn't expired (`reservation-expired`, via `common.NewExpiredError`) — a failed confirm never mutates status.
- **`Appointment.Cancel()`** only allows `Confirmed` → `Cancelled` (`appointment-not-confirmed` otherwise) and rejects appointments whose start time has passed.
- **"Expired" is not a stored status.** It's computed on read via `Appointment.IsExpired()` (`pending && startTime already passed`) — this is a no-show safety net independent of any specific `Reservation`'s own `IsExpired()` (the short checkout-window expiry checked inside `Confirm`). Nothing ever persists an "expired" value to the DB status column.
- `domain.AppointmentRepository` (in `domain/appointment.go`) is the outbound port: `RequestAppointment(ctx, userID, vehicle, *Appointment, *Reservation) error`, `ConfirmAppointment(ctx, userID, ReservationUUID) (AppointmentUUID, error)`, `CancelAppointment(ctx, userID, AppointmentUUID) error`. No implementation exists yet — this is pure interface, waiting on an adapter.
