// Package factory provides common errors.
package factory

import "errors"

var (
	// ErrInvalidConfig is returned when configuration type is invalid.
	ErrInvalidConfig = errors.New("invalid configuration type")

	// ErrDriverNotRegistered is returned when a driver is not registered.
	ErrDriverNotRegistered = errors.New("driver not registered")
)
