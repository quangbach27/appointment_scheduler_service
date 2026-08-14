package testutils

import "scheduler/internal/common"

// RandomString returns a fresh UUIDv7 string, unique per call — the shared
// building block test fixtures use for user IDs, idempotency keys, and
// other ad hoc uniqueness needs.
func RandomString() string {
	return common.NewUUIDv7().String()
}

// RandomID returns a prefixed identifier, e.g. RandomID("dealer") ->
// "dealer-0198f2e3-...", readable in DB rows and logs.
func RandomID(prefix string) string {
	return prefix + "-" + RandomString()
}
