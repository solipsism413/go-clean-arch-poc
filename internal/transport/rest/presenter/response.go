// Package presenter provides HTTP response formatting utilities.
package presenter

import (
	"encoding/json"
	"net/http"

	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
)

// Response represents a successful API response.
type Response struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
	Meta    any  `json:"meta,omitempty"`
}

// ErrorResponse represents an error API response.
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// FieldError represents a field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// JSON writes a successful JSON response.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := Response{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// JSONWithMeta writes a successful JSON response with metadata.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}

	json.NewEncoder(w).Encode(response)
}

// Error writes an error JSON response.
func Error(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	code := statusToCode(status)

	response := ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// ValidationError writes a validation error response.
func ValidationError(w http.ResponseWriter, validationErr *validation.ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	details := make([]FieldError, 0, len(validationErr.Errors))
	for _, e := range validationErr.Errors {
		details = append(details, FieldError{
			Field:   e.Field,
			Message: e.Message,
		})
	}

	response := ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    "VALIDATION_ERROR",
			Message: "Validation failed",
			Details: details,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// statusToCode converts HTTP status code to error code string.
func statusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "ERROR"
	}
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Created writes a 201 Created response with data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// OK writes a 200 OK response with data.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}
