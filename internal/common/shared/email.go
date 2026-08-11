// Package shared contains shared helper types and functions
// used across the application.
package shared

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type Email struct {
	email string
}

func NewEmail(email string) (Email, error) {
	sanitizedEmail := strings.TrimSpace(email)

	if !emailRegex.MatchString(sanitizedEmail) {
		return Email{}, errors.New("invalid email")
	}

	return Email{
		email: sanitizedEmail,
	}, nil
}

func (e Email) IsZero() bool {
	return e == Email{}
}

func (e Email) String() string {
	return e.email
}

// Scan implements the database/sql.Scanner interface, so pgx (via
// database/sql or pgx.Rows.Scan) can populate Email directly from a query.
func (e *Email) Scan(src any) error {
	if src == nil {
		return nil
	}

	switch v := src.(type) {
	case string:
		e.email = v
	case []byte:
		e.email = string(v)
	default:
		return fmt.Errorf("email: unsupported Scan type %T", src)
	}

	return nil
}

// Value implements the database/sql/driver.Valuer interface
func (e Email) Value() (driver.Value, error) {
	return e.email, nil
}
