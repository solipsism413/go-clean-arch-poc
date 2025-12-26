package valueobject_test

import (
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with dots", "test.user@example.com", false},
		{"valid email with plus", "test+tag@example.com", false},
		{"valid email uppercase", "TEST@EXAMPLE.COM", false},
		{"valid email with numbers", "test123@example.com", false},
		{"empty email", "", true},
		{"no at sign", "testexample.com", true},
		{"no domain", "test@", true},
		{"no local part", "@example.com", true},
		{"invalid domain", "test@.com", true},
		{"spaces", "test @example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := valueobject.NewEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && email.IsEmpty() {
				t.Error("NewEmail() should not return empty email for valid input")
			}
		})
	}
}

func TestEmail_Normalization(t *testing.T) {
	email, err := valueobject.NewEmail("  TEST@EXAMPLE.COM  ")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	if email.String() != "test@example.com" {
		t.Errorf("Email should be normalized to lowercase, got %v", email.String())
	}
}

func TestEmail_Domain(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")

	if domain := email.Domain(); domain != "example.com" {
		t.Errorf("Domain() = %v, want example.com", domain)
	}
}

func TestEmail_LocalPart(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")

	if localPart := email.LocalPart(); localPart != "test" {
		t.Errorf("LocalPart() = %v, want test", localPart)
	}
}

func TestEmail_Equals(t *testing.T) {
	email1, _ := valueobject.NewEmail("test@example.com")
	email2, _ := valueobject.NewEmail("TEST@EXAMPLE.COM") // Should be normalized
	email3, _ := valueobject.NewEmail("other@example.com")

	if !email1.Equals(email2) {
		t.Error("Emails with same normalized value should be equal")
	}

	if email1.Equals(email3) {
		t.Error("Different emails should not be equal")
	}
}

func TestEmail_Value(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")

	if email.Value() != "test@example.com" {
		t.Errorf("Value() = %v, want test@example.com", email.Value())
	}
}
