// Package factory provides common errors.
package factory

import "errors"

var (
	// ErrInvalidConfig is returned when configuration type is invalid.
	ErrInvalidConfig = errors.New("invalid configuration type")

	// ErrDriverNotFound is returned when a driver is not found.
	ErrDriverNotFound = errors.New("driver not found")
)
