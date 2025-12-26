package valueobject

import (
	"regexp"
	"strings"
)

// Email represents a validated email address.
type Email struct {
	value string
}

// emailRegex is a simple regex for email validation.
// For production, consider using a more comprehensive validation.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NewEmail creates a new Email value object.
// Returns an error if the email is invalid.
func NewEmail(email string) (Email, error) {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	if email == "" {
		return Email{}, ErrEmptyEmail
	}

	if !emailRegex.MatchString(email) {
		return Email{}, ErrInvalidEmail
	}

	return Email{value: email}, nil
}

// String returns the string representation of the email.
func (e Email) String() string {
	return e.value
}

// Value returns the underlying email string.
func (e Email) Value() string {
	return e.value
}

// Domain returns the domain part of the email.
func (e Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// LocalPart returns the local part (before @) of the email.
func (e Email) LocalPart() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

// Equals checks if two emails are equal.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// IsEmpty returns true if the email is empty.
func (e Email) IsEmpty() bool {
	return e.value == ""
}
